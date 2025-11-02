---
description: Regenerate sqlc database code after schema or query changes
---

Regenerate type-safe Go code from SQL queries using sqlc.

Steps:
1. Navigate to database directory: `cd internal/controlplane/database`
2. Run sqlc generation: `sqlc generate`
3. Return to project root: `cd ../../..`
4. Check for generated files: `ls -la internal/controlplane/database/generated/`
5. Report success and list generated files
6. Remind to commit generated code along with schema/query changes
