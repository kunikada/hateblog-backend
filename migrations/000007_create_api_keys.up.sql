-- Create api_keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash TEXT NOT NULL,
    name VARCHAR(100),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_ip INET,
    created_user_agent TEXT,
    created_referrer TEXT,

    -- Constraints
    CONSTRAINT api_keys_key_hash_unique UNIQUE (key_hash),
    CONSTRAINT api_keys_expires_at_check CHECK (expires_at IS NULL OR expires_at > created_at)
);

-- Create indexes
CREATE INDEX idx_api_keys_is_active ON api_keys (is_active);
CREATE INDEX idx_api_keys_expires_at ON api_keys (expires_at);
CREATE INDEX idx_api_keys_last_used_at ON api_keys (last_used_at DESC);

-- Add comment
COMMENT ON TABLE api_keys IS 'API認証キーの管理テーブル。キーのハッシュ値と作成時のメタデータを保存';
COMMENT ON COLUMN api_keys.id IS 'APIキーID（主キー）';
COMMENT ON COLUMN api_keys.key_hash IS 'APIキーのハッシュ値（bcryptまたはargon2）';
COMMENT ON COLUMN api_keys.name IS 'キーの名前（管理用）';
COMMENT ON COLUMN api_keys.description IS 'キーの説明';
COMMENT ON COLUMN api_keys.created_at IS 'レコード作成日時';
COMMENT ON COLUMN api_keys.expires_at IS '有効期限（NULLの場合は無期限）';
COMMENT ON COLUMN api_keys.last_used_at IS '最終使用日時';
COMMENT ON COLUMN api_keys.is_active IS '有効/無効フラグ';
COMMENT ON COLUMN api_keys.created_ip IS '作成時のIPアドレス';
COMMENT ON COLUMN api_keys.created_user_agent IS '作成時のUser-Agent';
COMMENT ON COLUMN api_keys.created_referrer IS '作成時のReferrer';
