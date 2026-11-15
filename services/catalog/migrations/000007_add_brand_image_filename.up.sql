ALTER TABLE brands
    ADD COLUMN image_filename varchar(255),
    ADD CONSTRAINT brands_image_filename_requires_key
        CHECK (image_filename IS NULL OR image_object_key IS NOT NULL);
