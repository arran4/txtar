# txtar CLI Skill for AI Agents

`txtar` is a command-line tool for managing txtar format archives. An agent can use it to inspect, create, update, or remove files packed into single-file txtar representations.

## Key Features
- **create**: Create archives. Supports recursive `-r`, following symlinks `-f`, and max depth `--depth`. Default behavior captures single files, but directories must be recursive.
- **list**: List files in an archive.
- **add/append**: Add files to an existing archive (modifies in place).
- **delete**: Delete files from an archive. Note: accepts glob patterns like `*.go`.
- **cat**: Extract/display file content from an archive.
- **comment**: View or set the archive comment via `--comment` string or `--file` path.

## Pitfalls & Tips for AI Agents
- The CLI manipulates txtar files **in-place** for destructive commands like `add` and `delete`. Always back up original txtar files if you need a recovery point.
- When creating archives containing directories, you MUST pass `-r` / `--recursive`.
- To avoid including unrelated files, you should utilize the `--name` glob filter with `create`.
- The CLI relies strictly on Go's `filepath.Match` for globbing (so `**` is not supported; use standard `*` and `?`).
- Output from `list` contains indices, offsets, sizes, and file names, which is highly useful for parsing specific byte ranges of the archive safely.
