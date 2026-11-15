ALTER TABLE brands
    DROP CONSTRAINT brands_image_filename_requires_key,
    DROP COLUMN image_filename;
