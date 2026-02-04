.PHONY: setup dev run-be run-fe

setup:
	@echo "🔧 Installing all dependencies..."
	cd backend && go mod tidy
	cd frontend && npm install

dev:
	@echo "🚀 Starting Overture Services..."
	make -j 2 run-be run-fe

run-be:
	cd backend && go run cmd/api/main.go

run-fe:
	cd frontend && npm run dev
