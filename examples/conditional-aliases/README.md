# Conditional Aliases Example

This example demonstrates how to use conditional aliases in Dirvana. Conditional aliases allow you to execute different commands based on runtime conditions like file existence, environment variables, or available commands.

## Features

### Basic Conditions

#### File Condition
Check if a file exists before executing a command:

```yaml
config:
  when:
    file: "config.json"
  command: cat config.json
  else: cp config.template.json config.json && cat config.json
```

#### Variable Condition
Verify an environment variable is set and non-empty:

```yaml
aws-deploy:
  when:
    var: "AWS_PROFILE"
  command: aws deploy push
  else: echo "Error: AWS_PROFILE environment variable is not set"
```

#### Directory Condition
Check if a directory exists:

```yaml
build:
  when:
    dir: "node_modules"
  command: npm run build
  else: echo "Error: node_modules not found. Run 'npm install' first"
```

#### Command Condition
Verify a command is available in PATH:

```yaml
docker-build:
  when:
    command: "docker"
  command: docker build -t myapp .
  else: echo "Error: Docker is not installed"
```

### Multiple Conditions

#### All Conditions (AND logic)
All conditions must be true:

```yaml
kubectl:
  when:
    all:
      - var: "KUBECONFIG"
      - file: "$KUBECONFIG"
  command: kubectl --kubeconfig "$KUBECONFIG"
  else: kubectl
```

#### Any Conditions (OR logic)
At least one condition must be true:

```yaml
config-edit:
  when:
    any:
      - file: ".env.local"
      - file: ".env"
  command: ${EDITOR:-vim} $([ -f .env.local ] && echo .env.local || echo .env)
  else: echo "Error: No config file found"
```

#### Nested Conditions
Combine `all` and `any` for complex logic:

```yaml
start:
  when:
    all:
      - dir: "node_modules"
      - any:
          - file: ".env.local"
          - file: ".env"
  command: npm start
  else: echo "Error: Run 'npm install' and create a .env file first"
```

## How It Works

1. **Condition Evaluation**: When you execute an alias, Dirvana evaluates the `when` conditions
2. **Success Path**: If conditions pass, the `command` is executed
3. **Fallback Path**: If conditions fail and `else` is defined, the fallback command runs
4. **Error Path**: If conditions fail and no `else` is defined, an error message shows which conditions failed

## Environment Variable Expansion

File and directory paths support environment variable expansion using `$VAR` or `${VAR}` syntax:

```yaml
kubectl:
  when:
    all:
      - var: "KUBECONFIG"  # Check KUBECONFIG is set
      - file: "$KUBECONFIG"  # Check the file at $KUBECONFIG path exists
  command: kubectl --kubeconfig "$KUBECONFIG"
  else: kubectl
```

## Error Messages

When conditions fail and no `else` is specified, you'll see descriptive error messages:

```
$ prod-deploy
Error: condition not met for alias 'prod-deploy':
  - environment variable 'PROD_API_KEY' is not set or empty
  - file 'dist/bundle.js' does not exist
```

## Testing This Example

1. Navigate to this directory:
   ```bash
   cd examples/conditional-aliases
   ```

2. Test with Dirvana:
   ```bash
   # This will fail because config.json doesn't exist (will use fallback)
   dirvana exec config

   # This will fail with error (no AWS_PROFILE set)
   dirvana exec aws-deploy

   # Set AWS_PROFILE and try again
   export AWS_PROFILE=default
   dirvana exec aws-deploy
   ```

3. Create test files to see conditions pass:
   ```bash
   touch config.json
   mkdir node_modules
   touch .env
   ```

## Best Practices

1. **Use meaningful fallbacks**: Provide helpful error messages or alternative commands
2. **Check dependencies**: Verify tools are installed before using them
3. **Validate environment**: Check required environment variables are set
4. **Fail fast**: For critical operations, omit `else` to show explicit errors
5. **Document conditions**: Use comments to explain why conditions are needed
