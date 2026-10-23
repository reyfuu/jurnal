# RAG infrastructure implementation plan

**Goal:** Prepare incremental ingestion of the 1,512-entry regulation catalog on the VPS, indexed retrieval, a grounded answer API, and lossless PDF derivatives.
**Architecture:** Existing PostgreSQL plus GIN full-text search. A separate, sequential worker reads index.csv and local PDF/HTML files; hashes content and metadata, commits one document and its page chunks atomically, and retries missing/failed files on later runs. No corpus download, VPS deployment, paid embedding calls, or changes to original files.
**Tech stack:** Go, existing pgx/PDF reader/pdfcpu, PostgreSQL, golang.org/x/net/html; separate worker image with a static Go binary.
**Spec:** User approved preparing infrastructure locally while completing the VPS document folder later.

## Constraints and decisions
- Work directly in pajak as requested; preserve existing untracked installation artifacts. No worktree or commits.
- Backend infrastructure only; no frontend redesign. Semantic embedding/model selection remains a later decision; do not require pgvector on the existing postgres:17-alpine instance.
- Documents are shared official regulations, not tenant uploads. Apply the same authentication, role, and subscription gates as Primbon.
- Ready documents only in retrieval. Preserve unknown, amended, and repealed status in citations; do not infer legal validity.
- Index per page with bounded chunks; checksums include extraction version and metadata. Missing files and OCR requirements are visible, never silently treated as indexed.
- Compression creates validated derivatives outside the source folder, compares page count and text, and keeps only smaller outputs.
- Replace legacy knowledge full-table rewrites with per-row updates based on old/new state; keep existing persistence interfaces for startup imports.

## Tasks
1. `internal/rag`: schema, chunking, safe catalog paths, bounded PDF/HTML extraction, atomic ingestion, GIN retrieval and statistics. Test idempotence, changed and missing sources, rollback, Unicode, page citations, and path escapes.
2. `cmd/rag`, `Dockerfile.rag`: dry-run by default; explicit apply, resumable per-document processing, timeouts, JSON progress and compression output. Test missing files, OCR, compression rejection, originals intact.
3. `cmd/server/rag.go`: protected search/status and question routes, bounded queries/context, deterministic no-key response, grounded model prompt and allowlisted references. Test existing auth/subscription boundaries and no-evidence behavior.
4. Legacy knowledge persistence: update/delete changed rows only; verify unchanged row tuples remain unchanged and failures roll back.
5. Run focused RED/GREEN tests, full Go suite, SQL integration when test DB is available, build server and worker. Add runbook and handoff with measured checks and explicit remaining VPS steps.

## Review focus
- Traversal/symlink escape and malicious metadata never expose arbitrary files or URLs.
- Incomplete corpus and extraction errors leave actionable statuses, not fabricated answers.
- Repeated imports do not rewrite chunks; changed imports atomically replace old evidence.
- No knowledge crosses existing access boundaries; document content cannot choose citations.
- Compression never rewrites original or accepts larger, invalid, or text-changing derivatives.

## Execution ledger
- Approved scope: infrastructure now, corpus on VPS completed later.
- LeanToken MCP not exposed in this session; CLI doctor failed first retrieval with access denied to its index. Using bounded native reads, not claiming LeanToken indexing succeeded.
- Ruling: reuse the existing native PDF reader and pdfcpu instead of adding Poppler/qpdf executables. Real PDF tests prove page/text preservation and unchanged originals; the worker needs no external PDF tools.
- Completed all five tasks. Native PostgreSQL 17 tests pass, including all existing TestPostgres tests and the new delta/RAG integration tests. Windows full suite has exactly two pre-existing Unix-permission failures, reproduced against HEAD; all remaining tests pass with those two explicitly skipped.
- Linux amd64 server and worker builds pass. Full catalog dry-run: missing=1512, failed=0; no source PDFs or production database touched.
- Fresh read-only review found an OCR branch skipping compression. Reproduced with a failing importer test, fixed, test passes; reviewer verified the correction.
- No harness rule changes proposed: isolated worker/resource-limit guidance and Windows baseline caveat belong in docs/RAG.md rather than expanding global agent rules.
