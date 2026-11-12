ALTER TABLE brands ADD CONSTRAINT brands_name_length_check
    CHECK (CHAR_LENGTH(name) <= 100);
