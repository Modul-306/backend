# Backend API Service

A Go-based REST API service that provides endpoints for managing users, blogs, products, and orders. Built with PostgreSQL for data persistence and JWT for authentication.

## 🛠 Tech Stack

- **Language**: Go 1.23
- **Database**: PostgreSQL
- **Libraries**:
  - `github.com/gorilla/mux` - HTTP router
  - `github.com/jackc/pgx/v5` - PostgreSQL driver
  - `github.com/golang-jwt/jwt` - JWT authentication
  - `golang.org/x/crypto` - Password hashing

## 📦 Project Structure

```
.
├── auth/           # Authentication related code
├── cmd/            # Application entrypoint
├── db/            # Database models and queries
├── handlers/      # HTTP request handlers
└── sql/          # SQL schema and queries
```

## 🗄 Database Schema

| Table | Description |
|-------|-------------|
| users | User accounts with authentication details |
| blogs | Blog posts created by users |
| products | Available products in the system |
| orders | User orders with delivery information |
| order_products | Many-to-many relationship between orders and products |

## 🔑 API Endpoints

### Authentication
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/auth/login` | User login | No |
| POST | `/api/v1/auth/sign-up` | User registration | No |

### Users
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/user` | List all users | Yes |
| GET | `/api/v1/user/{id}` | Get user details | Yes |
| UPDATE | `/api/v1/user/{id}` | Update user | Yes |
| DELETE | `/api/v1/user/{id}` | Delete user | Yes |

### Blogs
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/blogs` | List all blogs | No |
| GET | `/api/v1/blogs/{id}` | Get blog details | No |
| POST | `/api/v1/blogs` | Create new blog | Yes |
| UPDATE | `/api/v1/blogs/{id}` | Update blog | Yes |
| DELETE | `/api/v1/blogs/{id}` | Delete blog | Yes |

### Products
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/products` | List all products | No |
| GET | `/api/v1/products/{id}` | Get product details | No |
| POST | `/api/v1/products` | Create new product | Yes |
| UPDATE | `/api/v1/products/{id}` | Update product | Yes |
| DELETE | `/api/v1/products/{id}` | Delete product | Yes |

### Orders
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/order` | List all orders | Yes |
| GET | `/api/v1/order/{id}` | Get order details | Yes |
| POST | `/api/v1/order` | Create new order | Yes |
| UPDATE | `/api/v1/order/{id}` | Update order | Yes |
| DELETE | `/api/v1/order/{id}` | Delete order | Yes |

## 🚀 Getting Started

### Prerequisites
- Go 1.23+
- PostgreSQL
- Docker (optional)

### Environment Variables
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=testdb
```

### Running the Application

1. Clone the repository
2. Set up the database:
```bash
./test.sh  # This will set up a test database with sample data
```

3. Run the application:
```bash
go build -o main ./cmd/main.go
./main
```

The server will start on port 8000.

## 🧪 Testing

The repository includes a test script (`test.sh`) that:
- Sets up a PostgreSQL container
- Creates the database schema
- Inserts test data
- Runs the application
- Performs a test request

Run tests with:
```bash
./test.sh
```

## 📄 License

This project is licensed under the MIT License - see the 
LICENSE file for details.
