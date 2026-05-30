CREATE TABLE transfers (
    id uuid NOT NULL,
    timestamp timestamp NOT NULL,
    from_account_id uuid NOT NULL references accounts(id),
    to_account_id uuid NOT NULL references accounts(id),
    amount decimal NOT NULL,
    PRIMARY KEY (id)
)
