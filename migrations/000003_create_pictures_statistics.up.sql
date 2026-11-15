-- Pictures: the gallery. Rows describe objects stored in S3; the database keeps
-- only metadata and the public URLs, exactly as the legacy backend did.
CREATE TABLE IF NOT EXISTS pictures
(
    id                BIGSERIAL PRIMARY KEY,
    filename          VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type         VARCHAR(100) NOT NULL,
    size              BIGINT       NOT NULL,
    url               VARCHAR(500) NOT NULL,
    thumbnail_url     VARCHAR(500),
    alt               VARCHAR(500),
    uploader_id       VARCHAR(255),
    show_in_gallery   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pictures_show_in_gallery ON pictures (show_in_gallery);
CREATE INDEX IF NOT EXISTS idx_pictures_created_at ON pictures (created_at DESC);

CREATE TABLE IF NOT EXISTS picture_tags
(
    picture_id BIGINT NOT NULL REFERENCES pictures (id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (picture_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_picture_tags_tag_id ON picture_tags (tag_id);

-- Site statistics: one row per snapshot date. The landing page reads the latest.
CREATE TABLE IF NOT EXISTS site_statistics
(
    id                  BIGSERIAL PRIMARY KEY,
    total_articles      BIGINT      NOT NULL DEFAULT 0,
    total_categories    BIGINT      NOT NULL DEFAULT 0,
    total_tags          BIGINT      NOT NULL DEFAULT 0,
    total_article_views BIGINT      NOT NULL DEFAULT 0,
    date                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_site_statistics_date ON site_statistics (date DESC);
