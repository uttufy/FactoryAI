# FactoryAI Formulas Guide

## Overview

Formulas are TOML-based workflow recipes that define Standard Operating Procedures (SOPs) for FactoryAI. They specify how work should flow through stations, including dependencies between steps, roles, and templates.

## Formula Structure

### Basic Formula

```toml
name = "Feature Implementation"
description = "Standard workflow for implementing new features"
version = "1.0"

# Global variables available to all steps
[variables]
feature_branch_prefix = "feature/"
review_threshold = 2

[[steps]]
name = "design"
role = "Architect"
description = "Create technical design"
template = "Design a solution for: {task}\nConsider scalability and maintainability."
max_retries = 2
timeout = "30m"

[[steps]]
name = "implement"
role = "Developer"
description = "Implement the feature"
depends_on = ["design"]
template = """
Implement the feature based on this design:

{context}

Requirements:
- Write tests
- Update documentation
- Follow coding standards
"""
max_retries = 3
timeout = "1h"

[[steps]]
name = "review"
role = "Reviewer"
description = "Code review"
depends_on = ["implement"]
template = "Review the implementation:\n\n{context}\n\nCheck for correctness, style, and security."
max_retries = 1

[[steps]]
name = "merge"
role = "Integrator"
description = "Merge to main"
depends_on = ["review"]
template = "Merge the reviewed code:\n\n{context}"
```

## Formula Elements

### Metadata

```toml
name = "Formula Name"           # Required: Human-readable name
description = "Description"     # Required: What this formula does
version = "1.0"                 # Optional: Version string
author = "Team Name"            # Optional: Creator
```

### Variables

Define reusable variables:

```toml
[variables]
project_name = "myproject"
default_branch = "main"
timeout = "30m"
```

Use in templates: `{variables.project_name}`

### Steps

Each step represents one unit of work:

```toml
[[steps]]
name = "step-name"              # Required: Unique identifier
role = "Role Name"              # Required: Operator role
description = "What this does"  # Optional: Human-readable
depends_on = ["step1", "step2"] # Optional: Dependencies
template = "Prompt template"    # Required: Work prompt
max_retries = 3                 # Optional: Retry count (default: 1)
timeout = "30m"                 # Optional: Time limit
priority = 10                   # Optional: Execution priority
```

### Dependencies

Steps can depend on multiple previous steps:

```toml
[[steps]]
name = "final"
depends_on = ["step1", "step2", "step3"]
template = "Combine results from all previous steps"
```

The step will only execute after ALL dependencies complete successfully.

## Template Variables

### Built-in Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{task}` | Original user task | "Implement feature X" |
| `{context}` | Output from previous steps | Design document content |
| `{role}` | Current step's role | "Architect" |
| `{step}` | Current step name | "design" |
| `{station}` | Station ID | "station-1" |
| `{bead}` | Bead ID | "bead-123" |

### Variables from Previous Steps

Access output from specific steps:

```toml
[[steps]]
name = "design"
template = "Create design for: {task}"

[[steps]]
name = "implement"
depends_on = ["design"]
template = """
Implement based on design:

{steps.design.output}

Also consider: {task}
"""
```

### Global Variables

Access formula variables:

```toml
[variables]
api_endpoint = "https://api.example.com"

[[steps]]
template = "Call API at: {variables.api_endpoint}"
```

## Dependency Patterns

### Linear (Sequential)

```toml
[[steps]]
name = "step1"

[[steps]]
name = "step2"
depends_on = ["step1"]

[[steps]]
name = "step3"
depends_on = ["step2"]
```

Execution: `step1 → step2 → step3`

### Diamond (Merge)

```toml
[[steps]]
name = "start"

[[steps]]
name = "branch_a"
depends_on = ["start"]

[[steps]]
name = "branch_b"
depends_on = ["start"]

[[steps]]
name = "merge"
depends_on = ["branch_a", "branch_b"]
template = """
Combine:
A: {steps.branch_a.output}

B: {steps.branch_b.output}
"""
```

Execution: `start → [branch_a, branch_b] → merge`

### Fan-out (Parallel)

```toml
[[steps]]
name = "setup"

[[steps]]
name = "test_1"
depends_on = ["setup"]

[[steps]]
name = "test_2"
depends_on = ["setup"]

[[steps]]
name = "test_3"
depends_on = ["setup"]
```

Execution: `setup → [test_1, test_2, test_3]` (parallel)

### Complex DAG

```toml
[[steps]]
name = "design"

[[steps]]
name = "backend"
depends_on = ["design"]

[[steps]]
name = "frontend"
depends_on = ["design"]

[[steps]]
name = "backend_tests"]
depends_on = ["backend"]

[[steps]]
name = "integration_test"
depends_on = ["backend", "frontend"]
```

Execution: `design → [backend, frontend] → backend_tests → [frontend, integration_test]`

## Advanced Features

### Conditional Execution

```toml
[[steps]]
name = "deploy"
template = """
{%- if steps.test.output == "PASS" -%}
Deploy to production
{%- else -%}
Tests failed, skipping deploy
{%- endif -%}
"""
```

### Error Handling

```toml
[[steps]]
name = "risky_step"
max_retries = 3
on_failure = "fallback"
template = "Attempt risky operation"

[[steps]]
name = "fallback"
depends_on = ["risky_step"]
template = "Handle failure gracefully"
```

### Timeouts

```toml
[[steps]]
name = "long_running"
timeout = "2h"
template = "Long running task"
```

### Priority

```toml
[[steps]]
name = "critical"
priority = 100  # Higher priority

[[steps]]
name = "optional"
priority = 1   # Lower priority
```

## Example Formulas

### Code Review Workflow

```toml
name = "Code Review"
description = "Multi-perspective code review"

[[steps]]
name = "correctness"
role = "Senior Engineer"
description = "Review for correctness"
template = """
Review the following code for correctness:

{task}

Focus on:
- Logic errors
- Edge cases
- Race conditions
- Resource leaks
"""

[[steps]]
name = "style"
role = "Style Reviewer"
description = "Review for style"
template = """
Review the following code for style:

{task}

Focus on:
- Naming conventions
- Code organization
- Comments
- Documentation
"""

[[steps]]
name = "security"
role = "Security Expert"
description = "Review for security"
template = """
Review the following code for security issues:

{task}

Focus on:
- Input validation
- Authentication
- Authorization
- Secrets handling
- SQL injection
- XSS vulnerabilities
"""

[[steps]]
name = "performance"
role = "Performance Engineer"
description = "Review for performance"
template = """
Review the following code for performance:

{task}

Focus on:
- Algorithmic complexity
- Database queries
- Caching strategies
- Memory usage
"""

[[steps]]
name = "summary"
role = "Lead Engineer"
depends_on = ["correctness", "style", "security", "performance"]
template = """
Summarize the code review:

Correctness: {steps.correctness.output}
Style: {steps.style.output}
Security: {steps.security.output}
Performance: {steps.performance.output}

Provide:
1. Overall assessment
2. Must-fix issues
3. Nice-to-have improvements
4. Approval decision
"""
```

### Release Workflow

```toml
name = "Release"
description = "Complete software release workflow"

[variables]
version_bump = "patch"
changelog_file = "CHANGELOG.md"

[[steps]]
name = "test"
role = "QA Engineer"
description = "Run full test suite"
template = """
Run comprehensive tests for release: {task}

Tests to run:
- Unit tests
- Integration tests
- End-to-end tests
- Performance tests
"""
timeout = "1h"

[[steps]]
name = "bump_version"
role = "Release Manager"
depends_on = ["test"]
description = "Bump version number"
template = """
Bump version ({variables.version_bump}) based on: {context}

Update:
- Version in code
- Package.json/go.mod
- Documentation
"""

[[steps]]
name = "update_changelog"
role = "Technical Writer"
depends_on = ["bump_version"]
description = "Update changelog"
template = """
Update {variables.changelog_file} based on: {context}

Include:
- New features
- Bug fixes
- Breaking changes
- Migration notes
"""

[[steps]]
name = "tag"
role = "Release Manager"
depends_on = ["update_changelog"]
description = "Create git tag"
template = """
Create and push git tag for: {context}

Tag format: v{version}
"""

[[steps]]
name = "build"
role = "Build Engineer"
depends_on = ["tag"]
description = "Build release artifacts"
template = """
Build release artifacts for: {context}

Platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)
"""

[[steps]]
name = "deploy_staging"
role = "DevOps Engineer"
depends_on = ["build"]
description = "Deploy to staging"
template = """
Deploy to staging environment: {context}

Steps:
1. Upload artifacts
2. Run smoke tests
3. Verify deployment
"""
timeout = "30m"

[[steps]]
name = "smoke_test"
role = "QA Engineer"
depends_on = ["deploy_staging"]
description = "Run smoke tests"
template = """
Run smoke tests on staging: {context}

Tests:
- Basic functionality
- API endpoints
- Database connections
"""

[[steps]]
name = "deploy_production"
role = "DevOps Engineer"
depends_on = ["smoke_test"]
description = "Deploy to production"
template = """
Deploy to production: {context}

Steps:
1. Create backup
2. Deploy artifacts
3. Run verification
4. Monitor metrics
"""
```

### Bug Fix Workflow

```toml
name = "Bug Fix"
description = "Standard bug fix workflow"

[[steps]]
name = "reproduce"
role = "QA Engineer"
description = "Reproduce the bug"
template = """
Reproduce the bug: {task}

Provide:
1. Steps to reproduce
2. Expected behavior
3. Actual behavior
4. Error messages/logs
"""

[[steps]]
name = "diagnose"
role = "Senior Engineer"
depends_on = ["reproduce"]
description = "Diagnose root cause"
template = """
Diagnose the bug based on: {context}

Provide:
1. Root cause analysis
2. Affected components
3. Risk assessment
4. Recommended fix approach
"""

[[steps]]
name = "fix"
role = "Developer"
depends_on = ["diagnose"]
description = "Implement fix"
template = """
Implement fix for: {context}

Based on diagnosis:
{steps.diagnose.output}

Requirements:
- Minimal code changes
- Add tests for the bug
- Update documentation
"""

[[steps]]
name = "verify_fix"
role = "QA Engineer"
depends_on = ["fix"]
description = "Verify the fix"
template = """
Verify the fix: {context}

Tests:
1. Reproduce original bug (should fail)
2. Test fix (should pass)
3. Regression tests
4. Edge case testing
"""

[[steps]]
name = "review"
role = "Senior Engineer"
depends_on = ["verify_fix"]
description = "Review fix"
template = """
Review bug fix: {context}

Check:
- Fix is correct
- No side effects
- Tests are adequate
- Documentation updated
"""
```

## Best Practices

### 1. Keep Steps Focused

Each step should do one thing well:

```toml
# Bad: Does multiple things
[[steps]]
name = "do_everything"
template = "Design, implement, test, and deploy"

# Good: Single responsibility
[[steps]]
name = "design"
template = "Design the solution"

[[steps]]
name = "implement"
depends_on = ["design"]
template = "Implement the design"

[[steps]]
name = "test"
depends_on = ["implement"]
template = "Test the implementation"
```

### 2. Use Descriptive Names

```toml
# Bad
[[steps]]
name = "step1"
[[steps]]
name = "step2"

# Good
[[steps]]
name = "design_database"
[[steps]]
name = "implement_schema"]
```

### 3. Specify Timeouts

```toml
[[steps]]
name = "long_task"
timeout = "2h"  # Prevent indefinite hangs
```

### 4. Set Appropriate Retries

```toml
# External API calls: More retries
[[steps]]
name = "api_call"
max_retries = 5

# Manual review: Fewer retries
[[steps]]
name = "manual_review"
max_retries = 1
```

### 5. Document Dependencies

```toml
[[steps]]
name = "final_integration"
depends_on = ["backend_complete", "frontend_complete", "tests_pass"]
# Clear that all three must complete first
```

## Creating Custom Formulas

1. **Identify the workflow** - Map out the steps and dependencies
2. **Create TOML file** - In `formulas/` directory
3. **Define steps** - With clear roles and templates
4. **Test locally** - Use `./factory formula load`
5. **Iterate** - Refine based on results

Example file structure:

```
formulas/
├── feature.toml
├── bugfix.toml
├── release.toml
├── code-review.toml
└── custom/
    ├── team-specific.toml
    └── project-specific.toml
```

## Troubleshooting

### Formula Won't Load

```bash
# Check syntax
./factory formula validate --path ./formulas/myformula.toml

# View details
./factory formula status <formula-id>
```

### Steps Not Executing

- Check dependencies are satisfied
- Verify station availability
- Check operator status
- Review event logs

### Template Variables Not Working

- Ensure variable names match exactly
- Use correct syntax: `{variable_name}`
- Check for typos in template

## See Also

- [Architecture Documentation](ARCHITECTURE.md)
- [CLI Reference](CLI.md)
- [Event System](EVENTS.md)
