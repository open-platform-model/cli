# Experiments

Standalone experiments for validating CLI design decisions before integration.

Each experiment is a self-contained proof-of-concept that tests a specific architectural hypothesis or implementation approach for the OPM CLI.

## Writing an Experiment

### 1. Naming Convention

Use sequential three-digit index prefix:

```text
NNN-descriptive-name/
```

Examples: `005-policy-evaluation/`, `006-diff-engine/`

### 2. Required Structure

Each experiment MUST include:

| File | Purpose |
|------|---------|
| `README.md` | Goal, architecture, running instructions, test commands |
| `go.mod` (if Go) | Go module definition |
| `main.go` (if Go) | Entry point |
| `cue.mod/` (if CUE-only) | CUE module definition |

### 3. README Template

Every experiment README MUST include:

```markdown
# NNN-experiment-name

Brief one-line description.

## Goal

What architectural question does this experiment answer?

## Running

\`\`\`bash
cd cli/experiments/NNN-experiment-name

# Basic run
go run .

# With flags (if applicable)
go run . -v
\`\`\`

## Testing

\`\`\`bash
# How to validate the experiment works
go run . | kubectl apply --dry-run=client -f -

# Or run unit tests
go test ./...
\`\`\`

## Files

| File | Purpose |
|------|---------|
| `main.go` | ... |
| `pkg/` | ... |
```

### 4. Organization Guidelines

**Self-Contained Dependencies:**

- Prefer local `pkg/` or `testdata/` over registry imports for rapid iteration
- Document external dependencies (registry setup, Docker, etc.) clearly
- Include setup scripts (`scripts/setup.sh`) when external services are required

**Clear Success Criteria:**

- Define what "working" looks like (output format, test commands)
- Include example output or validation steps
- Document expected vs. actual behavior

**Findings & Comparison:**

- Record what you learned, even if the approach was rejected
- Compare to alternative approaches (see `003-hybrid-render/README.md` comparison tables)
- Reference relevant spec sections the experiment validates

## Running Tests

Each experiment defines its own test commands. Common patterns:

```bash
# Run the experiment
cd cli/experiments/NNN-name
go run .

# Verbose mode (most experiments support)
go run . -v

# Validate output (if producing K8s manifests)
go run . | kubectl apply --dry-run=client -f -

# Unit tests (if applicable)
go test ./...
```

## Existing Experiments

| # | Name | Purpose | Status |
|---|------|---------|--------|
| 002 | cue-based-matching | CUE-only transformer matching logic | Complete |
| 003 | hybrid-render | Go orchestration + CUE matching | ✓ Recommended |
| 004 | oci-module-loading | OCI registry + path overlays | Complete |

**Note:** Experiment 002 predates the README template and may lack full documentation.

## When to Graduate

Consider moving an experiment to the main CLI when:

1. **Validation complete** - The approach is proven and recommended
2. **Spec alignment** - Implementation matches accepted spec requirements
3. **Test coverage** - Includes unit tests and integration tests
4. **Performance validated** - Benchmarks confirm acceptable performance
5. **Team consensus** - Architecture review approves the approach

See `003-hybrid-render` as an example of a validated experiment ready for integration.
