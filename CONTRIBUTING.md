# Contributing to K8s Cost Optimizer

Thank you for your interest in contributing!

## Development Setup

1. **Prerequisites**
```bash
   - Go 1.21+
   - Docker
   - kubectl
   - minikube (for local testing)
```

2. **Clone & Build**
```bash
   git clone https://github.com/opscart/k8s-cost-optimizer.git
   cd k8s-cost-optimizer
   go build ./...
```

3. **Run Tests**
```bash
   go test ./...
```

## Project Structure

- `cmd/` - CLI applications
- `pkg/` - Reusable packages
- `docs/` - Documentation
- `examples/` - Sample configurations
- `scripts/` - Development scripts

## Making Changes

1. **Create a branch**
```bash
   git checkout -b feature/your-feature-name
```

2. **Make your changes**
   - Write clean, documented code
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**
```bash
   go test ./...
   go build ./...
```

4. **Commit with clear messages**
```bash
   git commit -m "feat: add new feature"
   git commit -m "fix: resolve issue with scanner"
   git commit -m "docs: update README"
```

5. **Push and create PR**
```bash
   git push origin feature/your-feature-name
```

## Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test changes
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write meaningful variable names
- Add comments for complex logic
- Keep functions focused and small

## Testing

- Write unit tests for new functionality
- Ensure existing tests pass
- Add integration tests for major features

## Documentation

- Update README.md for user-facing changes
- Update architecture docs for design changes
- Add inline code comments
- Update database schema docs if applicable

## Questions?

Open an issue or discussion on GitHub!
