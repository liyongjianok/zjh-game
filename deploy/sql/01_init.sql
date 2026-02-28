-- deploy/sql/01_init.sql
CREATE DATABASE IF NOT EXISTS zjh_core DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE zjh_core;

-- 玩家核心资产表
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '全局唯一玩家ID',
    `username` VARCHAR(64) NOT NULL COMMENT '登录账号',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希',
    `nickname` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '游戏内昵称',
    `coin_balance` BIGINT NOT NULL DEFAULT 0 COMMENT '金币余额(分为单位，避免浮点数精度问题)',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1:正常 0:封禁',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username` (`username`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '玩家基础信息与资产表';

-- 资产流水账单表（用于T+1对账）
CREATE TABLE IF NOT EXISTS `transaction_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `amount` BIGINT NOT NULL COMMENT '变更金额(正负值)',
    `tx_type` TINYINT NOT NULL COMMENT '1:充值 2:对局赢取 3:对局输掉 4:系统抽水',
    `reference_id` VARCHAR(128) NOT NULL COMMENT '关联的对局ID或订单ID',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_time` (`user_id`, `created_at`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '资产变更流水表';