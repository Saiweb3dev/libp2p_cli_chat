package p2p

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "math"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p/core/peer"
)

const (
    TaskProtocolID = "/task/1.0.0"
)

// TaskType represents different types of computational tasks
type TaskType string

const (
    TaskTypePrime     TaskType = "prime"
    TaskTypeFactorial TaskType = "factorial"
    TaskTypeFibonacci TaskType = "fibonacci"
    TaskTypeSum       TaskType = "sum"
)

// Task represents a computational task to be executed
type Task struct {
    ID       string      `json:"id"`
    Type     TaskType    `json:"type"`
    Number   int64       `json:"number,omitempty"`
    Numbers  []int64     `json:"numbers,omitempty"`
    Priority int         `json:"priority"`
    Created  time.Time   `json:"created"`
}

// TaskResult represents the result of a completed task
type TaskResult struct {
    TaskID    string        `json:"task_id"`
    Result    interface{}   `json:"result"`
    Success   bool          `json:"success"`
    Error     string        `json:"error,omitempty"`
    Duration  time.Duration `json:"duration"`
    WorkerID  string        `json:"worker_id"`
    Completed time.Time     `json:"completed"`
}

// TaskMessage wraps task communication
type TaskMessage struct {
    Type    string      `json:"type"` // "task" or "result"
    Task    *Task       `json:"task,omitempty"`
    Result  *TaskResult `json:"result,omitempty"`
}

// TaskService handles distributed task processing
type TaskService struct {
    host        host.Host
    chatService *ChatService
    
    // Task queue for workers
    taskQueue    []Task
    queueMu      sync.Mutex
    
    // Pending tasks for coordinators
    pendingTasks map[string]*PendingTask
    pendingMu    sync.RWMutex
    
    // Worker status
    isWorker     bool
    isCoordinator bool
    
    // Task processing
    processing   sync.WaitGroup
    shutdown     chan struct{}
}

// PendingTask tracks tasks waiting for completion
type PendingTask struct {
    Task      Task
    StartTime time.Time
    WorkerID  peer.ID
}

// NewTaskService creates a new task processing service
func NewTaskService(h host.Host, cs *ChatService, isWorker, isCoordinator bool) *TaskService {
    ts := &TaskService{
        host:          h,
        chatService:   cs,
        taskQueue:     make([]Task, 0),
        pendingTasks:  make(map[string]*PendingTask),
        isWorker:      isWorker,
        isCoordinator: isCoordinator,
        shutdown:      make(chan struct{}),
    }
    
    // Register task protocol handler
    h.SetStreamHandler(TaskProtocolID, ts.handleTaskStream)
    
    // Start worker if enabled
    if isWorker {
        go ts.workerLoop()
    }
    
    return ts
}

// handleTaskStream processes incoming task messages
func (ts *TaskService) handleTaskStream(s network.Stream) {
    defer s.Close()
    
    peerID := s.Conn().RemotePeer()
    reader := bufio.NewReader(s)
    
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                log.Printf("📪 Task stream closed by: %s", peerID)
            } else {
                log.Printf("❌ Error reading task message: %v", err)
            }
            return
        }
        
        var msg TaskMessage
        if err := json.Unmarshal([]byte(line), &msg); err != nil {
            log.Printf("❌ Error parsing task message: %v", err)
            continue
        }
        
        switch msg.Type {
        case "task":
            if msg.Task != nil && ts.isWorker {
                ts.queueTask(*msg.Task)
                log.Printf("📋 Received task %s from %s", msg.Task.ID, ts.chatService.GetPeerName(peerID))
            }
        case "result":
            if msg.Result != nil && ts.isCoordinator {
                ts.handleTaskResult(*msg.Result)
            }
        }
    }
}

// queueTask adds a task to the worker's queue
func (ts *TaskService) queueTask(task Task) {
    ts.queueMu.Lock()
    defer ts.queueMu.Unlock()
    
    // Insert task based on priority (higher priority first)
    inserted := false
    for i, existingTask := range ts.taskQueue {
        if task.Priority > existingTask.Priority {
            ts.taskQueue = append(ts.taskQueue[:i], append([]Task{task}, ts.taskQueue[i:]...)...)
            inserted = true
            break
        }
    }
    
    if !inserted {
        ts.taskQueue = append(ts.taskQueue, task)
    }
}

// workerLoop continuously processes tasks from the queue
func (ts *TaskService) workerLoop() {
    log.Printf("🔄 Worker started, processing tasks...")
    
    for {
        select {
        case <-ts.shutdown:
            return
        default:
            task := ts.dequeueTask()
            if task != nil {
                ts.processTask(*task)
            } else {
                time.Sleep(100 * time.Millisecond) // Wait if no tasks
            }
        }
    }
}

// dequeueTask removes and returns the highest priority task
func (ts *TaskService) dequeueTask() *Task {
    ts.queueMu.Lock()
    defer ts.queueMu.Unlock()
    
    if len(ts.taskQueue) == 0 {
        return nil
    }
    
    task := ts.taskQueue[0]
    ts.taskQueue = ts.taskQueue[1:]
    return &task
}

// processTask executes a task and sends the result back
func (ts *TaskService) processTask(task Task) {
    startTime := time.Now()
    log.Printf("⚙️  Processing task %s: %s", task.ID, task.Type)
    
    result := TaskResult{
        TaskID:    task.ID,
        WorkerID:  ts.host.ID().String(),
        Completed: time.Now(),
    }
    
    // Execute the task based on type
    switch task.Type {
    case TaskTypePrime:
        isPrime, err := ts.checkPrime(task.Number)
        result.Result = isPrime
        result.Success = err == nil
        if err != nil {
            result.Error = err.Error()
        }
        
    case TaskTypeFactorial:
        factorial, err := ts.calculateFactorial(task.Number)
        result.Result = factorial
        result.Success = err == nil
        if err != nil {
            result.Error = err.Error()
        }
        
    case TaskTypeFibonacci:
        fibonacci, err := ts.calculateFibonacci(task.Number)
        result.Result = fibonacci
        result.Success = err == nil
        if err != nil {
            result.Error = err.Error()
        }
        
    case TaskTypeSum:
        sum := ts.calculateSum(task.Numbers)
        result.Result = sum
        result.Success = true
        
    default:
        result.Success = false
        result.Error = fmt.Sprintf("unknown task type: %s", task.Type)
    }
    
    result.Duration = time.Since(startTime)
    
    // Send result back to all connected coordinators
    ts.sendResultToCoordinators(result)
}

// Task computation methods
func (ts *TaskService) checkPrime(n int64) (bool, error) {
    if n < 2 {
        return false, nil
    }
    if n == 2 {
        return true, nil
    }
    if n%2 == 0 {
        return false, nil
    }
    
    sqrt := int64(math.Sqrt(float64(n)))
    for i := int64(3); i <= sqrt; i += 2 {
        if n%i == 0 {
            return false, nil
        }
    }
    return true, nil
}

func (ts *TaskService) calculateFactorial(n int64) (int64, error) {
    if n < 0 {
        return 0, fmt.Errorf("factorial not defined for negative numbers")
    }
    if n > 20 { // Prevent overflow
        return 0, fmt.Errorf("factorial too large (max 20)")
    }
    
    result := int64(1)
    for i := int64(2); i <= n; i++ {
        result *= i
    }
    return result, nil
}

func (ts *TaskService) calculateFibonacci(n int64) (int64, error) {
    if n < 0 {
        return 0, fmt.Errorf("fibonacci not defined for negative numbers")
    }
    if n > 50 { // Prevent long computation
        return 0, fmt.Errorf("fibonacci sequence too long (max 50)")
    }
    
    if n <= 1 {
        return n, nil
    }
    
    a, b := int64(0), int64(1)
    for i := int64(2); i <= n; i++ {
        a, b = b, a+b
    }
    return b, nil
}

func (ts *TaskService) calculateSum(numbers []int64) int64 {
    sum := int64(0)
    for _, num := range numbers {
        sum += num
    }
    return sum
}

// sendResultToCoordinators sends task result to connected coordinators
func (ts *TaskService) sendResultToCoordinators(result TaskResult) {
    connectedPeers := ts.host.Network().Peers()
    
    msg := TaskMessage{
        Type:   "result",
        Result: &result,
    }
    
    data, err := json.Marshal(msg)
    if err != nil {
        log.Printf("❌ Error marshaling result: %v", err)
        return
    }
    
    for _, peerID := range connectedPeers {
        go func(pid peer.ID) {
            if err := ts.sendTaskMessage(pid, data); err != nil {
                log.Printf("❌ Error sending result to %s: %v", ts.chatService.GetPeerName(pid), err)
            }
        }(peerID)
    }
}

// SubmitTask submits a task to available workers (coordinator function)
func (ts *TaskService) SubmitTask(taskType TaskType, number int64, numbers []int64, priority int) (string, error) {
    if !ts.isCoordinator {
        return "", fmt.Errorf("this node is not configured as a coordinator")
    }
    
    taskID := fmt.Sprintf("task_%d_%s", time.Now().UnixNano(), taskType)
    task := Task{
        ID:       taskID,
        Type:     taskType,
        Number:   number,
        Numbers:  numbers,
        Priority: priority,
        Created:  time.Now(),
    }
    
    // Track pending task
    ts.pendingMu.Lock()
    ts.pendingTasks[taskID] = &PendingTask{
        Task:      task,
        StartTime: time.Now(),
    }
    ts.pendingMu.Unlock()
    
    // Send task to all connected workers
    connectedPeers := ts.host.Network().Peers()
    if len(connectedPeers) == 0 {
        return "", fmt.Errorf("no workers available")
    }
    
    msg := TaskMessage{
        Type: "task",
        Task: &task,
    }
    
    data, err := json.Marshal(msg)
    if err != nil {
        return "", fmt.Errorf("error marshaling task: %v", err)
    }
    
    sentCount := 0
    for _, peerID := range connectedPeers {
        if err := ts.sendTaskMessage(peerID, data); err != nil {
            log.Printf("❌ Error sending task to %s: %v", ts.chatService.GetPeerName(peerID), err)
        } else {
            sentCount++
        }
    }
    
    if sentCount == 0 {
        delete(ts.pendingTasks, taskID)
        return "", fmt.Errorf("failed to send task to any workers")
    }
    
    log.Printf("📤 Submitted task %s to %d workers", taskID, sentCount)
    return taskID, nil
}

// sendTaskMessage sends a task message to a specific peer
func (ts *TaskService) sendTaskMessage(peerID peer.ID, data []byte) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    s, err := ts.host.NewStream(ctx, peerID, TaskProtocolID)
    if err != nil {
        return err
    }
    defer s.Close()
    
    _, err = s.Write(append(data, '\n'))
    return err
}

// handleTaskResult processes completed task results
func (ts *TaskService) handleTaskResult(result TaskResult) {
    ts.pendingMu.Lock()
    pending, exists := ts.pendingTasks[result.TaskID]
    if exists {
        delete(ts.pendingTasks, result.TaskID)
    }
    ts.pendingMu.Unlock()
    
    if !exists {
        log.Printf("⚠️  Received result for unknown task: %s", result.TaskID)
        return
    }
    
    workerName := ts.chatService.GetPeerName(peer.ID(result.WorkerID))
    
    if result.Success {
        var resultStr string
        switch pending.Task.Type {
        case TaskTypePrime:
            isPrime := result.Result.(bool)
            if isPrime {
                resultStr = fmt.Sprintf("%d is prime", pending.Task.Number)
            } else {
                resultStr = fmt.Sprintf("%d is not prime", pending.Task.Number)
            }
        case TaskTypeFactorial:
            resultStr = fmt.Sprintf("factorial(%d) = %v", pending.Task.Number, result.Result)
        case TaskTypeFibonacci:
            resultStr = fmt.Sprintf("fibonacci(%d) = %v", pending.Task.Number, result.Result)
        case TaskTypeSum:
            resultStr = fmt.Sprintf("sum(%v) = %v", pending.Task.Numbers, result.Result)
        }
        
        log.Printf("✅ Task completed: %s (took %v, worker: %s)", 
            resultStr, result.Duration, workerName)
    } else {
        log.Printf("❌ Task failed: %s - %s (worker: %s)", 
            result.TaskID, result.Error, workerName)
    }
}

// GetQueueStatus returns current queue status
func (ts *TaskService) GetQueueStatus() (int, []Task) {
    ts.queueMu.Lock()
    defer ts.queueMu.Unlock()
    
    tasks := make([]Task, len(ts.taskQueue))
    copy(tasks, ts.taskQueue)
    return len(tasks), tasks
}

// GetPendingTasks returns pending tasks for coordinators
func (ts *TaskService) GetPendingTasks() map[string]*PendingTask {
    ts.pendingMu.RLock()
    defer ts.pendingMu.RUnlock()
    
    pending := make(map[string]*PendingTask)
    for id, task := range ts.pendingTasks {
        pending[id] = task
    }
    return pending
}

// Shutdown stops the task service
func (ts *TaskService) Shutdown() {
    close(ts.shutdown)
    ts.processing.Wait()
}