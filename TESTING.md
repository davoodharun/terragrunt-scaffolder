# Testing Guide

This document provides detailed information about testing the Terragrunt Scaffolder project.

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests for a specific package
go test ./internal/scaffold -v

# Run a specific test
go test ./internal/scaffold -v -run TestGenerateCommand
```

## Test Structure

The test suite is organized as follows:

```
.
├── internal/
│   ├── scaffold/
│   │   ├── scaffold_test.go    # Main scaffold package tests
│   │   ├── validate_test.go    # Validation logic tests
│   │   ├── environment_test.go # Environment generation tests
│   │   └── components_test.go  # Component generation tests
│   └── config/
│       └── config_test.go      # Configuration parsing tests
└── cmd/
    └── tgs/
        └── main_test.go        # CLI command tests
```

## Test Categories

1. **Unit Tests**
   - Test individual functions and components in isolation
   - Located in `internal/*/test.go` files
   - Focus on edge cases and error conditions

2. **Integration Tests**
   - Test the interaction between different components
   - Located in `internal/scaffold/scaffold_test.go`
   - Verify the complete generation process

3. **CLI Tests**
   - Test command-line interface functionality
   - Located in `cmd/tgs/main_test.go`
   - Verify command execution and output

## Writing Tests

When adding new features or fixing bugs, follow these testing guidelines:

1. **Test File Location**
   - Place test files next to the source files they test
   - Use the `_test.go` suffix
   - Follow the same package structure as the source

2. **Test Naming**
   - Use descriptive test names that explain the scenario
   - Follow the pattern: `Test{FunctionName}_{Scenario}`
   - Example: `TestValidateConfig_ValidConfiguration`

3. **Test Structure**
   ```go
   func TestFunctionName_Scenario(t *testing.T) {
       // Arrange
       // Set up test data and conditions

       // Act
       // Execute the function being tested

       // Assert
       // Verify the results
   }
   ```

4. **Table-Driven Tests**
   - Use table-driven tests for testing multiple scenarios
   - Example:
     ```go
     func TestValidateConfig(t *testing.T) {
         tests := []struct {
             name    string
             config  *config.MainConfig
             wantErr bool
         }{
             {
                 name:    "valid configuration",
                 config:  createValidConfig(),
                 wantErr: false,
             },
             {
                 name:    "missing project name",
                 config:  createInvalidConfig(),
                 wantErr: true,
             },
         }

         for _, tt := range tests {
             t.Run(tt.name, func(t *testing.T) {
                 err := validateConfig(tt.config)
                 if (err != nil) != tt.wantErr {
                     t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
                 }
             })
         }
     }
     ```

## Test Coverage

To generate and view test coverage:

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# View coverage in terminal
go tool cover -func=coverage.out
```

## Continuous Integration

Tests are automatically run in the CI pipeline for:
- Pull requests
- Merges to main branch
- Release tags

The CI pipeline ensures:
- All tests pass
- Code coverage meets minimum requirements
- No linting errors
- No security vulnerabilities

## Debugging Tests

To debug tests:

1. **Using Delve**
   ```bash
   # Install Delve
   go install github.com/go-delve/delve/cmd/dlv@latest

   # Run tests with Delve
   dlv test ./internal/scaffold -v -run TestGenerateCommand
   ```

2. **Using VS Code**
   - Set breakpoints in test files
   - Use the "Debug Test" option in the Testing sidebar
   - Use the debug console to inspect variables

## Best Practices

1. **Test Independence**
   - Each test should be independent
   - Don't rely on test execution order
   - Clean up any resources created during tests

2. **Meaningful Assertions**
   - Test specific outcomes, not implementation details
   - Use descriptive error messages
   - Include relevant context in failure messages

3. **Performance**
   - Keep tests fast and efficient
   - Use appropriate test helpers and mocks
   - Avoid unnecessary setup/teardown

4. **Maintainability**
   - Keep test code clean and well-organized
   - Use helper functions for common test setup
   - Document complex test scenarios 