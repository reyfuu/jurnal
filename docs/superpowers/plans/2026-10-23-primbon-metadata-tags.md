# Primbon tags and metadata

User approved implementation now, keeping the 6.6 GB corpus on the VPS. Work in the current pajak folder. Preserve existing visuals and uncommitted RAG work.

- Add optional metadata.json keyed by existing regulation UUID; metadata only, no naskah copying or automatic merging with the 20,798-entry reference catalog.
- Support topic tags, status/change, status-as-of date, verification date/source. Missing values remain unknown; metadata does not establish legal validity for arbitrary dates.
- Reload the catalog on index.csv or metadata.json changes; invalid updates retain the last good snapshot and are retried. Metadata works without PDFs present.
- Show consistent badges and effective/as-of information on catalog cards and detail, using existing tokens. Include tags in existing search.
- Tests first for metadata loading, missing PDFs, search, validation and reload. Verify Go tests, frontend build and rendered desktop/mobile where browser access permits. No VPS deployment or corpus download.
