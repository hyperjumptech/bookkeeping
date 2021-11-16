use wallet;

CREATE TABLE IF NOT EXISTS accounts (
  `account_number` VARCHAR(20) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `currency_code` VARCHAR(10) NOT NULL,
  `description` TEXT,
  `alignment` VARCHAR(6) NOT NULL,
  `balance` INT NOT NULL,
  `coa` VARCHAR(10),
  `created_at` TIMESTAMP,
  `created_by` VARCHAR(16),
  `updated_at` TIMESTAMP,
  `updated_by` VARCHAR(16),
  `is_deleted` TINYINT(1) DEFAULT false ,
  PRIMARY KEY (`account_number`),
  INDEX(`coa`, `name`)
);

CREATE TABLE IF NOT EXISTS currencies (
  `code` VARCHAR(10) NOT NULL,
  `name` VARCHAR(30) NOT NULL,
  `exchange` FLOAT NOT NULL,
  `created_at` TIMESTAMP,
  `created_by` VARCHAR(16),
  `updated_at` TIMESTAMP,
  `updated_by` VARCHAR(16),
  `is_deleted` TINYINT(1) DEFAULT false ,
  PRIMARY KEY (`code`)
);

CREATE TABLE IF NOT EXISTS journals (
  `journal_id` VARCHAR(20) NOT NULL,
  `journaling_time` TIMESTAMP NOT NULL,
  `description` TEXT,
  `is_reversal` TINYINT(1),
  `reversed_journal_id` VARCHAR(20),
  `total_amount` INT NOT NULL,
  `created_at` TIMESTAMP,
  `created_by` VARCHAR(16),
  `updated_at` TIMESTAMP,
  `updated_by` VARCHAR(16),
  `is_deleted` TINYINT(1) DEFAULT false ,
  PRIMARY KEY (`journal_id`)
);

CREATE TABLE IF NOT EXISTS transactions (
  `transaction_id` VARCHAR(20) NOT NULL,
  `account_number` VARCHAR(20) NOT NULL,
  `transaction_time` TIMESTAMP NOT NULL,
  `journal_id` VARCHAR(20) NOT NULL,
  `description` TEXT,
  `alignment` VARCHAR(6) NOT NULL,
  `amount` INT NOT NULL,
  `balance` INT NOT NULL,
  `created_at` TIMESTAMP,
  `created_by` VARCHAR(16),
  `updated_at` TIMESTAMP,
  `updated_by` VARCHAR(16),
  `is_deleted` TINYINT(1) DEFAULT false ,
  PRIMARY KEY (`transaction_id`),
  INDEX(`account_number`, `journal_id`)
);


