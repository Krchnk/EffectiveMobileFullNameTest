CREATE TYPE gender_type AS ENUM ('male', 'female', 'other');

CREATE TABLE persons (
                         id SERIAL PRIMARY KEY,
                         name VARCHAR(255) NOT NULL,
                         surname VARCHAR(255) NOT NULL,
                         patronymic VARCHAR(255),
                         age INTEGER,
                         gender gender_type,
                         nationality VARCHAR(50),
                         created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_persons_name ON persons (name);
CREATE INDEX idx_persons_surname ON persons (surname);