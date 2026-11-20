-- Convert the multilingual article/category schema to the single-language one:
-- translation content moves onto the base rows, then the translation tables go.

ALTER TABLE articles
    ADD COLUMN IF NOT EXISTS title VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS summary VARCHAR(500) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS markdown_content TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS html_content TEXT NOT NULL DEFAULT '';

-- Prefer the Chinese translation.
UPDATE articles AS a
SET title            = t.title,
    summary          = t.summary,
    markdown_content = t.markdown_content,
    html_content     = t.html_content
FROM article_translations AS t
WHERE t.article_id = a.id
  AND t.language = 'zh';

-- Fall back to the source translation when no Chinese one exists.
UPDATE articles AS a
SET title            = t.title,
    summary          = t.summary,
    markdown_content = t.markdown_content,
    html_content     = t.html_content
FROM article_translations AS t
WHERE t.article_id = a.id
  AND a.title = ''
  AND t.id = (
      SELECT choice.id
      FROM article_translations AS choice
      WHERE choice.article_id = a.id
      ORDER BY choice.is_source DESC, choice.id ASC
      LIMIT 1
  );

DROP TABLE IF EXISTS article_translations;

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';

-- Prefer the Chinese category name.
UPDATE categories AS c
SET name = ct.content
FROM category_translations AS ct
WHERE ct.category_id = c.id
  AND ct.language = 'zh';

-- Fall back to any translation, then to the slug.
UPDATE categories AS c
SET name = ct.content
FROM category_translations AS ct
WHERE ct.category_id = c.id
  AND c.name = ''
  AND ct.id = (
      SELECT choice.id
      FROM category_translations AS choice
      WHERE choice.category_id = c.id
      ORDER BY choice.id ASC
      LIMIT 1
  );

UPDATE categories SET name = slug WHERE name = '';

DROP TABLE IF EXISTS category_translations;

-- The base rows are now the source of truth; no silent empty defaults.
ALTER TABLE articles
    ALTER COLUMN title DROP DEFAULT,
    ALTER COLUMN summary DROP DEFAULT,
    ALTER COLUMN markdown_content DROP DEFAULT,
    ALTER COLUMN html_content DROP DEFAULT;

ALTER TABLE categories ALTER COLUMN name DROP DEFAULT;
