# Quickstart

Get Mesa up and running in five steps.

1. **Install Go** (1.26+ per go.mod): download from [go.dev/dl](https://go.dev/dl) or run `brew install go`

2. **Build the binary**:

   ```bash
   go build ./cmd/mesa
   ```

3. **Run it**:

   ```bash
   ./mesa
   ```

   Custom port: `./mesa 9090` or `PORT=3000 ./mesa`

4. **Open the dashboard** at [http://localhost:3001](http://localhost:3001) -- the default `PORT`. First run bootstraps a default org with 6 agents

5. **Submit your first issue**: create it on the dashboard board, or add an entry to `artifact-docs/backlog.md` and let the CEO agent triage it into issues
