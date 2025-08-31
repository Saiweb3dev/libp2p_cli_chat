#!/bin/bash
# filepath: /home/sai/Documents/Systems/p2p/libp2p_compute/start.sh

echo "🚀 Starting P2P Distributed Compute Network..."

# Build and start all services
docker-compose up --build -d

echo "⏳ Waiting for services to initialize..."
sleep 15

echo "📊 Network Status:"
echo "==================="

# Check bootstrap server
echo "🔗 Bootstrap Server:"
curl -s http://localhost:8080/health && echo " ✅ Healthy" || echo " ❌ Unhealthy"

# Show registered peers
echo -e "\n👥 Registered Peers:"
curl -s http://localhost:8080/peers | jq '.[] | "\(.name) (\(.role)) - \(.id)"' 2>/dev/null || echo "Unable to fetch peers"

echo -e "\n📋 Container Status:"
docker-compose ps

echo -e "\n🎯 Demo Tasks:"
echo "The coordinator will automatically start submitting demo tasks in ~10 seconds."
echo "Monitor the logs with: docker-compose logs -f coordinator worker1"

echo -e "\n📚 Useful Commands:"
echo "  View all logs:           docker-compose logs -f"
echo "  View coordinator logs:   docker-compose logs -f coordinator"
echo "  View worker logs:        docker-compose logs -f worker1 worker2 worker3 worker4"
echo "  Stop all services:       docker-compose down"
echo "  Interactive coordinator: docker-compose exec coordinator /app/bin/peer --name=manual-coord --coordinator --bootstrap=http://bootstrap:8080"