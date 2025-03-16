
# Backend API Service

A Go-based REST API service that provides endpoints for managing users, blogs, products, and orders. Built with PostgreSQL for data persistence and JWT for authentication.

## 🛠 Tech Stack

- **Language**: Go 1.23
- **Database**: PostgreSQL 14
- **Testing**: 
  - Testcontainers for PostgreSQL
  - Integration tests
  - Unit tests
- **Libraries**:
  - `github.com/gorilla/mux` - HTTP router
  - `github.com/jackc/pgx/v5` - PostgreSQL driver
  - `github.com/golang-jwt/jwt` - JWT authentication
  - `github.com/testcontainers/testcontainers-go` - Container testing

## 🚀 Getting Started

### Prerequisites
- Go 1.23+
- Docker
- Make (optional)

### Setup

1. Clone the repository:
```bash
git clone https://github.com/Modul-306/backend.git
cd backend
```

2. Install dependencies:
```bash
go mod download
```

3. Set environment variables:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=testdb
```

### Development

Run the application:
```bash
go run cmd/main.go
```

### Testing

The project uses testcontainers for integration testing:

```bash
# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./handlers
go test -v ./auth
```

Test features:
- Automated PostgreSQL container setup
- Schema initialization
- Test data seeding
- Cleanup after tests

### Project Structure
```
.
├── auth/           # Authentication
├── cmd/            # Application entrypoint
├── db/            # Database layer
├── handlers/      # HTTP handlers
├── sql/          # SQL schemas and queries
└── tests/        # Test utilities
    ├── containers/  # Test container setup
    └── testhelpers/ # Test helper functions
```

## 🧪 Testing Architecture

Tests are organized in multiple layers:

1. **Unit Tests**
   - Individual package functionality
   - No external dependencies

2. **Integration Tests**
   - Database operations
   - API endpoints
   - Uses testcontainers

3. **Test Helpers**
   - Database setup/cleanup
   - Test data generation
   - Container management

## 📝 Development Workflow

1. Write tests first
2. Implement features
3. Run test suite
4. Review code coverage
5. Submit PR

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Add tests
4. Implement changes
5. Submit pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
