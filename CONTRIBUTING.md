# Contributing to Tenet

Thank you for your interest in contributing to Tenet!

## Development Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/tenet.git
cd tenet

# Run tests
go test -v ./...

# Build CLI
go build -o tenet ./cmd/tenet

# Build WASM
GOOS=js GOARCH=wasm go build -o tenet.wasm ./wasm
```

## Project Structure

```
tenet/
├── schema.go        # Core types (Schema, Definition, Rule, etc.)
├── engine.go        # Run() and Verify() public APIs
├── resolver.go      # JSON-logic expression resolver
├── operators.go     # Operator implementations
├── temporal.go      # Temporal routing and pruning
├── validate.go      # Type and constraint validation
├── *_test.go        # Test files
├── cmd/tenet/       # CLI tool
├── wasm/            # WASM entry point
├── docs/            # Documentation
└── testdata/        # Test fixtures
```

## Running Tests

```bash
# All tests
go test -v ./...

# Specific test
go test -v -run TestReactiveTransformation ./...

# With coverage
go test -cover ./...
```

## Code Style

- Run `go fmt` before committing
- Add tests for new features
- Keep operators nil-safe (handle nil values gracefully)

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Add tests for your changes
4. Ensure all tests pass
5. Submit a pull request

## Adding New Operators

1. Add the operator case in `operators.go` in `executeOperator()`
2. Implement nil-safe logic (return appropriate default for nil inputs)
3. Add tests in `engine_test.go`
4. Document in `docs/README.md`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
