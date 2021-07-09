CREATE TABLE idn_award.accounts ( 
  `account_number` VARCHAR(20) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `currency_code` VARCHAR(3) NOT NULL,
  `description` TEXT,
  `alignment` VARCHAR(6) NOT NULL,
  `ballance` INT NOT NULL,
  `coa` VARCHAR(10),
  `created_at` TIMESTAMP,
  `created_by` VARCHAR(16),
  `updated_at` TIMESTAMP,
  `updated_by` VARCHAR(16),
  PRIMARY KEY (`account_number`),
  INDEX(`coa`)
 );

CREATE TABLE idn_award.currencies ( 
  `code` VARCHAR(3) NOT NULL,
  `name` VARCHAR(10) NOT NULL, 
  `exchange` FLOAT NOT NULL,
  `created_at` TIMESTAMP,
  `created_by` VARCHAR(16),
  `updated_at` TIMESTAMP,
  `updated_by` VARCHAR(16),
  PRIMARY KEY (`code`),
 );

CREATE TABLE idn_award.journals ( 
  `journal_id` VARCHAR(20) NOT NULL,
	`journaling_time` TIMESTAMP NOT NULL,
	`description` TEXT,
	`is_reversal` TINYINT(1),
	`reversed_jounal_id` INT,
	`total_amount` INT NOT NULL,
	`created_at` TIMESTAMP,
	`created_by` VARCHAR(16),
  PRIMARY KEY (`journal_id`),
 );

CREATE TABLE idn_award.transactions ( 
  `transaction_id` VARCHAR(20) NOT NULL,
  `account_number` VARCHAR(20) NOT NULL,
  `journal_id` VARCHAR(20) NOT NULL,
  `desc` TEXT,
  `alignment` VARCHAR(6) NOT NULL,
  `amount` INT NOT NULL,
  `balance` INT NOT NULL,
  `created_at` TIMESTAMP,
  `created_by` VARCHAR(16),
  PRIMARY KEY (`transaction_id`),
 );
