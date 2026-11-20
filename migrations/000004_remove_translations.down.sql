-- Restore the multilingual schema, carrying the single-language content back
-- into Chinese translation rows.

CREATE TABLE IF NOT EXISTS article_translations
(
    id                  BIGSERIAL PRIMARY KEY,
    article_id          BIGINT       NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    language            VARCHAR(2)   NOT NULL,
    title               VARCHAR(255) NOT NULL,
    summary             VARCHAR(255) NOT NULL,
    markdown_content    TEXT         NOT NULL,
    html_content        TEXT         NOT NULL,
    is_source           BOOLEAN      NOT NULL,
    is_translated_by_ai BOOLEAN      NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_article_translations_article_id ON article_translations (article_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_article_translations_article_languag ON article_translations (article_id, language);

INSERT INTO article_translations (article_id, language, title, summary, markdown_content, html_content, is_source, is_translated_by_ai)
SELECT id, 'zh', title, summary, markdown_content, html_content, TRUE, FALSE
FROM articles;

CREATE TABLE IF NOT EXISTS category_translations
(
    id          BIGSERIAL PRIMARY KEY,
    category_id BIGINT       NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    language    VARCHAR(2)   NOT NULL,
    content     VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_category_translations_category_id ON category_translations (category_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_category_translations_category_language
    ON category_translations (category_id, language);

INSERT INTO category_translations (category_id, language, content)
SELECT id, 'zh', name
FROM categories;

ALTER TABLE articles
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS summary,
    DROP COLUMN IF EXISTS markdown_content,
    DROP COLUMN IF EXISTS html_content;

ALTER TABLE categories DROP COLUMN IF EXISTS name;
