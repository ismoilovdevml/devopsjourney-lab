#!/bin/bash

echo "🚀 Starting DevOps Journey Lab..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose is not installed."
    exit 1
fi

echo "✅ Docker is running"
echo ""

# Build student lab image
echo "📦 Building student lab container image..."
docker build -f Dockerfile.simple -t devopslab:latest . -q

if [ $? -eq 0 ]; then
    echo "✅ Student lab image built successfully"
else
    echo "❌ Failed to build student lab image"
    exit 1
fi

echo ""

# Start services with docker-compose
echo "🐳 Starting services with Docker Compose..."
docker-compose up -d

if [ $? -eq 0 ]; then
    echo "✅ Services started successfully"
else
    echo "❌ Failed to start services"
    exit 1
fi

echo ""
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service health
echo ""
echo "🔍 Checking service health..."
docker-compose ps

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ DevOps Journey Lab is ready!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🌐 Access the application:"
echo "   Frontend:    http://localhost:3000"
echo "   Backend API: http://localhost:8080/api"
echo "   Health:      http://localhost:8080/healthz"
echo ""
echo "📊 View logs:"
echo "   docker-compose logs -f"
echo ""
echo "🛑 Stop services:"
echo "   docker-compose down"
echo ""
