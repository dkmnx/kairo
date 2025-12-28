# Claude Code Integration Examples

Practical examples of integrating Kairo with Claude Code for various workflows.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflows](#development-workflows)
- [Documentation Generation](#documentation-generation)
- [Code Review](#code-review)
- [Debugging Assistance](#debugging-assistance)
- [Testing Support](#testing-support)
- [DevOps Integration](#devops-integration)
- [AI-Powered Shell Scripts](#ai-powered-shell-scripts)

---

## Getting Started

### Basic Setup

```bash
# 1. Install Kairo
curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh

# 2. Setup a provider
kairo setup
# Choose your preferred provider (e.g., Z.AI, MiniMax, etc.)
# Enter your API key

# 3. Set default provider
kairo default zai

# 4. Test integration
kairo switch zai "Hello, can you hear me?"

# 5. Use direct query mode (default provider)
kairo -- "What's your model?"
kairo -- "Explain quantum computing"
```

### Understanding Query Modes

Kairo supports two query modes:

**1. Switch Mode** - Use specific provider:

```bash
kairo switch zai "Generate a Go function"
kairo switch minimax "Quick question"
```

**2. Direct Query Mode** - Use default provider with `--`:

```bash
# Set default once
kairo default zai

# Query without switching
kairo -- "What's your model?"
kairo -- "Explain this code"
kairo -- "Write a haiku"
```

**When to use `--` (Direct Query Mode):**

- Quick questions without specifying provider
- Shell aliases for common queries
- Scripts that need AI assistance
- Interactive terminal usage

**When to use `switch`:**

- Need specific provider capabilities
- Switching between providers for comparison
- One-time provider selection

### Verify Claude Code is Installed

```bash
# Check if Claude is available
claude --version

# Test basic query
claude "What is 2+2?"

# Configure Kairo provider
kairo default zai
```

---

## Development Workflows

### Scenario 1: Code Generation with Specific Provider

Generate code using different providers based on complexity:

```bash
# Simple tasks - use faster provider
kairo default minimax

kairo "Write a function to reverse a string in Python"

# Generate and save
kairo minimax "Write function" > reverse.py

# Complex tasks - use more capable provider
kairo switch zai "Implement a RESTful API in Go with Gin framework"
kairo switch zai "Create a React component with TypeScript and hooks"
```

### Scenario 2: Code Refactoring

Refactor code with provider-specific capabilities:

```bash
# Analyze code for refactoring opportunities
kairo switch zai "Analyze this Go code for refactoring opportunities:

package main

import 'fmt'

func main() {
    for i := 0; i < 10; i++ {
        fmt.Println(i)
        for j := 0; j < 10; j++ {
            fmt.Println(j)
        }
    }
}" > refactor_suggestions.md

# Apply suggested improvements
# Or ask Claude to apply directly
kairo switch zai "Refactor this Go code to use nested loops efficiently" < main.go > main_refactored.go
```

### Scenario 3: Multi-Language Code Translation

Translate code between languages:

```bash
# Python to Go
kairo switch zai "Convert this Python code to idiomatic Go:

def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)" > fibonacci.go

# JavaScript to Rust
kairo switch zai "Translate this JavaScript to Rust:

function debounce(func, wait) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}" > debounce.rs
```

### Scenario 4: Algorithm Implementation

Implement complex algorithms:

```bash
# Implement Dijkstra's algorithm
kairo switch zai "Implement Dijkstra's shortest path algorithm in Go with proper error handling and comments" > dijkstra.go

# Implement a neural network from scratch
kairo switch zai "Implement a simple neural network in Python with forward propagation and backpropagation" > neural_network.py

# Implement a caching solution
kairo switch zai "Implement an LRU cache in Go with O(1) operations" > lru_cache.go
```

### Scenario 5: Direct Query Mode with Shell Aliases

Use `--` for quick queries and set up shell aliases:

```bash
# Set default provider
kairo default zai

# Use direct query for quick questions
kairo -- "What's your model?"
kairo -- "Explain quantum computing"
kairo -- "Write a haiku about programming"

# Set up convenient aliases in ~/.bashrc or ~/.zshrc
# alias ai='kairo --'
# alias explain='kairo -- "Explain'
# alias debug='kairo -- "Debug'
# alias refactor='kairo -- "Refactor'
# alias test='kairo -- "Write tests for'

# Use in daily workflow
ai "What's the capital of France?"
explain "How does Kubernetes work?"
debug "Why is my API returning 500?"
refactor "this Go function for better performance"
test "the User class in user.py"

# Great for interactive development and quick assistance
function ai_assist() {
    kairo -- "$*"
}

# Usage
ai_assist "How do I reverse a string in Python?"
ai_assist "Explain the difference between map and reduce"
```

---

## Documentation Generation

### Scenario 1: API Documentation

Generate API documentation from code:

```bash
# Generate OpenAPI spec
kairo switch zai "Generate an OpenAPI 3.0 specification for this REST API:

package main

import (
    'github.com/gin-gonic/gin'
)

type User struct {
    ID    string \`json:'id'\`
    Name  string \`json:'name'\`
    Email string \`json:'email'\`
}

func setupRouter() *gin.Engine {
    r := gin.Default()
    r.GET('/users', getUsers)
    r.POST('/users', createUser)
    r.GET('/users/:id', getUser)
    r.DELETE('/users/:id', deleteUser)
    return r
}

func getUsers(c *gin.Context) {
    // Implementation
}

func createUser(c *gin.Context) {
    // Implementation
}

func getUser(c *gin.Context) {
    // Implementation
}

func deleteUser(c *gin.Context) {
    // Implementation
}" > openapi.yaml

# Generate README for API
kairo switch zai "Write a comprehensive README for this API with usage examples \
and authentication details" < openapi.yaml > API_README.md
```

### Scenario 2: Code Comments

Add comprehensive comments to code:

```bash
# Add Go doc comments
kairo switch zai "Add comprehensive godoc comments and package-level documentation to this Go code:

package cache

import (
    'sync'
    'time'
)

type Cache struct {
    items map[string]interface{}
    mu    sync.RWMutex
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
    // Implementation
}

func (c *Cache) Get(key string) (interface{}, bool) {
    // Implementation
}

func (c *Cache) Delete(key string) {
    // Implementation
}" > cache_documented.go
```

### Scenario 3: Architecture Documentation

Generate architecture documentation:

```bash
# Generate system architecture diagram description
kairo switch zai "Describe the microservices architecture for an e-commerce platform including:
1. User service
2. Product catalog service
3. Order service
4. Payment service
5. Notification service

Include communication patterns, data flow, and scalability considerations." > ARCHITECTURE.md

# Generate database schema documentation
kairo switch zai "Document this PostgreSQL schema with relationships, indexes, and constraints:

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    total DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);" > DATABASE_SCHEMA.md
```

---

## Code Review

### Scenario 1: Automated Code Review

Review pull requests with AI:

```bash
# Review Go code for best practices
kairo switch zai "Review this Go code for:
1. Go best practices and idiomatic patterns
2. Performance issues
3. Security vulnerabilities
4. Error handling
5. Code organization

Code:

package main

import (
    'fmt'
    'net/http'
)

func handler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get('q')
    fmt.Fprintf(w, 'Query: %s', query)
}

func main() {
    http.HandleFunc('/', handler)
    http.ListenAndServe(':8080', nil)
}" > code_review.md

# Generate pull request review comments
kairo switch zai "Generate constructive pull request review comments for the issues \
found in this code review" < code_review.md > pr_comments.md
```

### Scenario 2: Security Audit

Perform security audit:

```bash
# Audit for security issues
kairo switch zai "Perform a security audit on this code and highlight:
1. SQL injection vulnerabilities
2. XSS vulnerabilities
3. Authentication/authorization issues
4. Input validation problems
5. Dependency vulnerabilities

Code:

import sqlite3

def get_user(username):
    conn = sqlite3.connect('users.db')
    cursor = conn.cursor()
    query = 'SELECT * FROM users WHERE username = \'' + username + '\''
    cursor.execute(query)
    return cursor.fetchone()" > security_audit.md
```

### Scenario 3: Performance Analysis

Analyze performance bottlenecks:

```bash
# Analyze performance
kairo switch zai "Analyze this code for performance issues and suggest optimizations:

function processLargeArray(arr) {
    let result = [];
    for (let i = 0; i < arr.length; i++) {
        for (let j = 0; j < arr.length; j++) {
            result.push(arr[i] * arr[j]);
        }
    }
    return result;
}" > performance_analysis.md
```

---

## Debugging Assistance

### Scenario 1: Debug Production Issues

Debug real-world production issues:

```bash
# Debug memory leak
kairo switch zai "I have a Go application that's experiencing memory leaks. Here's the profile:

goroutine profile: total 1000
--- thread:
goroutine 1 [running]:
main.main()
    /app/main.go:45 +0x123

goroutine 2-999 [chan receive]:
net/http.(*conn).serve()
    /usr/local/go/src/net/http/server.go:3020 +0x456

Help me identify the cause and provide a fix." > debug_memory_leak.md

# Debug race condition
kairo switch zai "I'm seeing a race condition in my concurrent Go program. Here's the code:

var counter int

func increment() {
    counter++
}

func main() {
    for i := 0; i < 1000; i++ {
        go increment()
    }
    time.Sleep(time.Second)
    fmt.Println('Counter:', counter)
}

Explain the race condition and fix it with proper synchronization." > debug_race_condition.md
```

### Scenario 2: Error Diagnosis

Diagnose complex errors:

```bash
# Decode error messages
kairo switch zai "Help me understand and fix this error:

Error: runtime error: invalid memory address or nil pointer dereference
  at main.go:42:15
  called from main.main()
  goroutine 1 [running]

Code context:
func processUser(id int) *User {
    var user *User
    db.Query('SELECT * FROM users WHERE id = ?', id).Scan(&user)
    return user  // Line 42
}

Explain the issue and provide a corrected version." > error_diagnosis.md

# Debug API integration issues
kairo switch zai "I'm getting a 401 Unauthorized error when calling this API:

GET /api/users
Headers:
  Authorization: Bearer token123

Response:
401 Unauthorized
WWW-Authenticate: Bearer realm='api', error='invalid_token'

Help me debug the authentication flow." > api_debug.md
```

### Scenario 3: Log Analysis

Analyze application logs:

```bash
# Analyze error logs
kairo switch zai "Analyze these application logs and identify the root cause:

[ERROR] 2025-01-28 10:15:30 - Connection timeout: db.example.com:5432
[ERROR] 2025-1-28 10:15:35 - Failed to process order: connection refused
[WARN] 2025-01-28 10:16:00 - Database connection pool exhausted
[INFO] 2025-01-28 10:16:05 - Retrying connection...
[ERROR] 2025-01-28 10:16:10 - Connection refused: db.example.com:5432

Provide diagnosis and remediation steps." > log_analysis.md
```

---

## Testing Support

### Scenario 1: Generate Unit Tests

Generate comprehensive tests:

```bash
# Generate Go tests
kairo switch zai "Generate comprehensive unit tests for this Go function using testify:

func CalculateTax(price float64, rate float64) float64 {
    if price < 0 {
        return 0
    }
    if rate < 0 || rate > 1 {
        return 0
    }
    return price * rate
}

Include:
1. Happy path tests
2. Edge cases (negative values, zero)
3. Boundary conditions
4. Invalid inputs" > calculator_test.go

# Generate Python tests
kairo switch zai "Generate pytest tests for this Python class:

class ShoppingCart:
    def __init__(self):
        self.items = []

    def add_item(self, item, price):
        self.items.append((item, price))

    def remove_item(self, item):
        self.items = [i for i in self.items if i[0] != item]

    def total(self):
        return sum(price for _, price in self.items)

Include test cases for all methods" > shopping_cart_test.py
```

### Scenario 2: Generate Integration Tests

Create integration test scenarios:

```bash
# Generate API integration tests
kairo switch zai "Generate integration tests for a REST API using pytest-requests:

Test endpoints:
1. POST /users - Create user
2. GET /users/:id - Get user
3. PUT /users/:id - Update user
4. DELETE /users/:id - Delete user

Include:
1. Success scenarios
2. Validation errors
3. Authentication/authorization
4. Edge cases" > api_integration_test.py

# Generate database integration tests
kairo switch zai "Generate integration tests for this PostgreSQL schema:

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    stock INTEGER DEFAULT 0
);

Test:
1. CRUD operations
2. Foreign key constraints
3. Transaction rollback
4. Concurrent access" > database_integration_test.go
```

### Scenario 3: Test Data Generation

Generate test data:

```bash
# Generate synthetic test data
kairo switch zai "Generate realistic test data for an e-commerce platform with:
1. 100 users
2. 500 products
3. 200 orders
4. Proper relationships

Output as JSON and SQL INSERT statements." > test_data.json

# Generate edge case test data
kairo switch zai "Generate edge case test data for an address validation system:
1. Valid addresses from different countries
2. Invalid zip codes
3. Missing required fields
4. Special characters in fields
5. Extremely long values" > address_test_data.json
```

---

## DevOps Integration

### Scenario 1: CI/CD Pipeline

Integrate Kairo into CI/CD:

```yaml
# .github/workflows/test.yml
name: CI with AI Code Review
on: [pull_request]

jobs:
  code-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Kairo
        run: |
          curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh

      - name: Configure Provider
        env:
          ZAI_API_KEY: ${{ secrets.ZAI_API_KEY }}
        run: |
          kairo config zai

      - name: AI Code Review
        run: |
          git diff HEAD~1 > changes.diff
          kairo switch zai "Review these code changes for bugs, security issues, and \
best practices: $(cat changes.diff)" > ai_review.md

      - name: Upload Review
        uses: actions/upload-artifact@v3
        with:
          name: ai-review
          path: ai_review.md
```

### Scenario 2: Infrastructure as Code

Generate infrastructure code:

```bash
# Generate Terraform configuration
kairo switch zai "Generate Terraform configuration for:
1. AWS EC2 instances with auto-scaling
2. RDS PostgreSQL database
3. Application Load Balancer
4. CloudWatch monitoring

Include security groups, IAM roles, and tagging." > main.tf

# Generate Docker Compose
kairo switch zai "Generate a Docker Compose file for a web application with:
1. Nginx reverse proxy
2. Node.js application
3. PostgreSQL database
4. Redis cache
5. Proper networking and volumes" > docker-compose.yml
```

### Scenario 3: Monitoring and Alerting

Generate monitoring queries:

```bash
# Generate Prometheus queries
kairo switch zai "Generate Prometheus queries and alerts for monitoring a web application:
1. HTTP request rate
2. Response time percentiles
3. Error rate
4. Database connection pool
5. Memory usage" > prometheus.yml

# Generate Grafana dashboards
kairo switch zai "Generate a Grafana dashboard JSON configuration for monitoring:
1. CPU usage
2. Memory usage
3. Request throughput
4. Response times
5. Error rates" > dashboard.json
```

---

## AI-Powered Shell Scripts

### Scenario 1: Automated Git Workflow

```bash
#!/bin/bash
# git-helper.sh

case "$1" in
  commit)
    # Generate commit message
    MESSAGE=$(kairo switch zai "Write a concise git commit message for these changes: $(git diff --cached --stat)")
    git commit -m "$MESSAGE"
    ;;

  review)
    # Review changes
    kairo switch zai "Review this PR for issues: $(git diff HEAD~1..HEAD)"
    ;;

  deploy)
    # Check for issues before deploy
    ISSUES=$(kairo switch zai "Check this code for deployment issues: $(git diff HEAD~1)")
    if [ -z "$ISSUES" ]; then
      git push origin main
    else
      echo "Deployment blocked: $ISSUES"
    fi
    ;;
esac
```

### Scenario 2: Code Quality Checker

```bash
#!/bin/bash
# quality-check.sh

FILE=$1

# Check for anti-patterns
kairo switch zai "Check $FILE for code anti-patterns and refactoring opportunities"

# Check for security issues
kairo switch zai "Audit $FILE for security vulnerabilities"

# Suggest improvements
kairo switch zai "Suggest code quality improvements for $FILE"
```

### Scenario 3: Automated Documentation

```bash
#!/bin/bash
# generate-docs.sh

PROJECT=$1

# Generate README
kairo switch zai "Generate a comprehensive README.md for $PROJECT including:
1. Project overview
2. Installation instructions
3. Usage examples
4. API documentation
5. Contributing guidelines" > README.md

# Generate CHANGELOG
kairo switch zai "Generate a CHANGELOG.md based on recent git commits: $(git log --oneline -20)" > CHANGELOG.md
```

### Scenario 4: Interactive AI Assistant with Direct Query Mode

Create an interactive AI assistant using `--` for quick queries:

```bash
#!/bin/bash
# ai-assistant.sh

# Ensure default provider is set
kairo default zai

# Interactive AI assistant
ai_assist() {
    local query="$*"
    if [ -z "$query" ]; then
        # Read from stdin if no argument
        query=$(cat)
    fi
    kairo -- "$query"
}

# Create function for common tasks
explain() {
    kairo -- "Explain $*"
}

debug() {
    kairo -- "Debug: $*"
}

suggest() {
    kairo -- "Suggest improvements for: $*"
}

# Interactive mode
if [ "$1" == "--interactive" ]; then
    echo "Kairo AI Assistant (type 'exit' to quit)"
    while true; do
        read -p "AI> " query
        if [ "$query" == "exit" ]; then
            break
        fi
        kairo -- "$query"
    done
fi

# Use as wrapper script
case "$1" in
  explain|debug|suggest)
    $1 "${@:2}"
    ;;
  *)
    ai_assist "$@"
    ;;
esac
```

**Usage:**

```bash
# Set default provider once
kairo default zai

# Make script executable
chmod +x ai-assistant.sh

# Use functions
./ai-assistant.sh explain "How does Kubernetes work?"
./ai-assistant.sh debug "Why is my API returning 500?"
./ai-assistant.sh suggest "this Go code for better performance"

# Interactive mode
./ai-assistant.sh --interactive
```

**Add to .bashrc for global aliases:**

```bash
# Add to ~/.bashrc
source ~/path/to/ai-assistant.sh

# Now available globally
explain "Docker networking"
debug "This error message"
suggest "My Python function"

# Or use alias for direct queries
alias ai='kairo --'
ai "What's the weather like?"
ai "Convert this JSON to CSV"
```

---

## Best Practices

### 1. Choose the Right Provider

- **Z.AI**: General purpose, complex tasks, code generation
- **MiniMax**: Quick responses, simple tasks, cost optimization
- **Kimi**: Specialized tasks, multilingual
- **DeepSeek**: Cost-effective, batch processing
- **Custom**: Self-hosted, specific API requirements

### 2. Provide Context

```bash
# Good: Specific with context
kairo switch zai "Implement a REST API in Go using Gin framework for an e-commerce platform.
Requirements:
- CRUD operations for products
- Authentication middleware
- Rate limiting
- Logging

Use best practices and include error handling."

# Bad: Too vague
kairo switch zai "Write an API"
```

### 3. Iterative Development

```bash
# Step 1: Get initial implementation
kairo switch zai "Implement a binary search tree in Python"

# Step 2: Add features
kairo switch zai "Add deletion and traversal methods to the previous binary search tree"

# Step 3: Optimize
kairo switch zai "Optimize the binary search tree implementation for better performance"
```

### 4. Verify and Test

```bash
# Generate code
kairo switch zai "Implement a merge sort algorithm in Go" > merge_sort.go

# Test it
go test merge_sort.go

# Fix issues
kairo switch zai "Fix this error in merge_sort.go: $(go test merge_sort.go 2>&1)"
```

---

## Related Documentation

- [User Guide](user-guide.md)
- [Advanced Configuration](advanced-configuration.md)
- [Error Handling Examples](error-handling-examples.md)
- [Troubleshooting Guide](../troubleshooting/README.md)
