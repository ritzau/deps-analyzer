---
description: Commit and push changes with a structured template
---

1.  **Verify**: Ensure the code compiles and tests pass (if applicable for this atomic change).
    - Command: `go build ./... && go test ./...`
2.  **Add**: Stage the specific files for this atomic change.
    - Command: `git add <files>`
3.  **Commit**: Use the following template for the commit message:

    ```
    [Short Summary]

    Background:
    [Why is this change being made?]

    Change:
    [What was changed? Technical details.]

    Verified:
    [How was this verified? e.g. "Ran go test ./pkg/model", "Manual check of graph output"]
    ```

    - Command: `git commit -m "..."`

4.  **Push**: Push the changes immediately.
    - Command: `git push`
