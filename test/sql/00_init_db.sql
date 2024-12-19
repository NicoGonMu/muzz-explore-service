CREATE TABLE decisions
(
    actor_user_id VARCHAR(10) NOT NULL,
    recipient_user_id VARCHAR(10) NOT NULL,
    liked_recipient BOOLEAN NOT NULL,
    last_modified INT(11) NOT NULL,
    seen_by_recipient BOOLEAN NOT NULL,
    CONSTRAINT PK_decision PRIMARY KEY (actor_user_id,recipient_user_id)
);