CREATE TABLE IF NOT EXISTS rag_document (
 id TEXT PRIMARY KEY,
 kode TEXT NOT NULL,
 judul TEXT NOT NULL,
 status TEXT NOT NULL,
 berlaku TEXT NOT NULL,
 sumber_url TEXT NOT NULL,
 file TEXT NOT NULL,
 checksum TEXT NOT NULL DEFAULT '',
 state TEXT NOT NULL CHECK (state IN ('ready','missing','needs_ocr','error')),
 error TEXT NOT NULL DEFAULT '',
 metadata JSONB NOT NULL DEFAULT '{}',
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE rag_document ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';
CREATE TABLE IF NOT EXISTS rag_chunk (
 document_id TEXT NOT NULL REFERENCES rag_document(id) ON DELETE CASCADE,
 ordinal INT NOT NULL,
 page INT NOT NULL CHECK (page > 0),
 text TEXT NOT NULL CHECK (length(text) > 0),
 search TSVECTOR NOT NULL,
 PRIMARY KEY (document_id, ordinal)
);
CREATE INDEX IF NOT EXISTS rag_chunk_search ON rag_chunk USING GIN(search);
CREATE INDEX IF NOT EXISTS rag_document_state ON rag_document(state);
