-- 暗号鍵テーブルの作成
CREATE TABLE IF NOT EXISTS encryption_keys (
    id CHAR(36) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    generation INT UNSIGNED NOT NULL,
    encrypted_key BLOB NOT NULL,
    status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_tenant_generation (tenant_id, generation),
    INDEX idx_tenant_id (tenant_id),
    INDEX idx_tenant_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
