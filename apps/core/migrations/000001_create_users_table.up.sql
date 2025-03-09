CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_auth_providers (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- e.g., "google"
    display_name VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    hd VARCHAR(255),
    picture TEXT, -- Profile picture URL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, provider) -- Ensures each user has only one entry per provider
);
