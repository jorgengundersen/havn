- claim the issue
- resolve the issue following @specs/code-standards.md
- remember red/green TDD. One test, one implementation (no batch)
- when job is done, commit the work to trigger pre-commit hook
- When work is successfully committed and pushed, add any important notes to the beads (if relevant and valuable), and close the issue
- Then exit

if you encounter a bug:
1. unrelated to current work: file with `bd create --type=bug`, continue current work
2. related and straightforward fix: file with `bd create --type=bug`, fix it, update the bug issue with `bd note <id> "fix details"`, close it, continue current work

if the issue is unclear or needs human review:
- `bd note <id> "expected X but found Y"`
- `bd label add <id> human`
- `bd update <id> --status=open --assignee=""`
- Then exit
