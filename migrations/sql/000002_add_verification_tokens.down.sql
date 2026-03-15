DROP TABLE IF EXISTS verification_tokens;
 
ALTER TABLE users REMOVE IF EXISTS is_verified;