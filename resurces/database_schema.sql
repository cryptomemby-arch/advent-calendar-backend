-- Database schema for Advent Calendar Backend
-- This file contains CREATE TABLE statements for all necessary tables
-- and initial data inserts.

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    avatar VARCHAR(255),
    streak INTEGER DEFAULT 0 NOT NULL,
    total_points BIGINT DEFAULT 0 NOT NULL,
    theme_preference VARCHAR(255) DEFAULT 'SYSTEM' NOT NULL,
    auth_provider VARCHAR(255),
    auth_subject VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add check constraint for theme_preference
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_theme_preference_check;
ALTER TABLE users ADD CONSTRAINT users_theme_preference_check CHECK (theme_preference IN ('LIGHT', 'DARK', 'SYSTEM'));

-- Challenges table
CREATE TABLE IF NOT EXISTS challenges (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,
    energy_level VARCHAR(50) NOT NULL,
    active BOOLEAN DEFAULT FALSE NOT NULL,
    culture VARCHAR(50) DEFAULT 'GLOBAL' NOT NULL
);

-- User challenges table
CREATE TABLE IF NOT EXISTS user_challenges (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    challenge_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    mood VARCHAR(20),
    assigned_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (challenge_id) REFERENCES challenges(id) ON DELETE CASCADE
);

-- Add check constraint for status
ALTER TABLE user_challenges DROP CONSTRAINT IF EXISTS user_challenges_status_check;
ALTER TABLE user_challenges ADD CONSTRAINT user_challenges_status_check CHECK (status IN ('ASSIGNED', 'COMPLETED'));

-- Time capsules table
CREATE TABLE IF NOT EXISTS time_capsules (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    reveal_date TIMESTAMP NOT NULL,
    revealed BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Photos table
CREATE TABLE IF NOT EXISTS photos (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    public_id VARCHAR(255) NOT NULL,
    secure_url VARCHAR(255) NOT NULL,
    caption TEXT,
    format VARCHAR(50),
    width INTEGER,
    height INTEGER,
    bytes BIGINT,
    taken_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_user_challenges_user_id ON user_challenges(user_id);
CREATE INDEX IF NOT EXISTS idx_user_challenges_challenge_id ON user_challenges(challenge_id);
CREATE INDEX IF NOT EXISTS idx_time_capsules_user_id ON time_capsules(user_id);
CREATE INDEX IF NOT EXISTS idx_photos_user_id ON photos(user_id);
CREATE INDEX IF NOT EXISTS idx_photos_created_at ON photos(created_at);

-- Initial challenges data
INSERT INTO challenges (title, description, category, energy_level, active, culture) VALUES
('Hidden Cafe Discovery', 'Find a quiet cafe you have never visited and spend 30 minutes there reading or people-watching.', 'EXPLORE_CITY', 'LOW', true, 'GLOBAL'),
('Street Art Snapshot', 'Walk one street you rarely take and photograph 3 pieces of street art or murals.', 'EXPLORE_CITY', 'LOW', true, 'GLOBAL'),
('Park Bench Pause', 'Visit a nearby park you do not usually go to and sit for 20 minutes observing the area.', 'EXPLORE_CITY', 'LOW', true, 'GLOBAL'),
('Market Mystery Tour', 'Visit a local market and try one food item you have never tasted.', 'EXPLORE_CITY', 'MEDIUM', true, 'GLOBAL'),
('Historic Block Fact Walk', 'Pick a historic building, walk there, and learn one fact from a plaque or a quick search.', 'EXPLORE_CITY', 'MEDIUM', true, 'GLOBAL'),
('Spice Aisle Hunt', 'Go to a grocery store and find three Indian spices or ingredients you have not used before.', 'EXPLORE_CITY', 'MEDIUM', true, 'INDIA'),
('Transit Adventure', 'Take a bus or train line you have never used and explore the final stop for at least 1 hour.', 'EXPLORE_CITY', 'HIGH', true, 'GLOBAL'),
('Rangoli Pattern Walk', 'Walk around campus or your city and photograph 5 colorful geometric patterns inspired by rangoli.', 'EXPLORE_CITY', 'HIGH', true, 'INDIA'),
('City Square Loop', 'Do a 30-minute loop around your city center or main plaza, inspired by Russian city squares, and note three landmarks.', 'EXPLORE_CITY', 'HIGH', true, 'RUSSIA'),
('Geometry Architecture Walk', 'Take a long walk and photograph 5 bold geometric building features inspired by constructivist design.', 'EXPLORE_CITY', 'HIGH', true, 'RUSSIA'),
('Trend Research Hour', 'Spend an hour exploring a current student trend you do not understand and learn what people enjoy about it.', 'TREND_BASED', 'LOW', true, 'GLOBAL'),
('Campus Meme Museum', 'Collect five campus memes or inside jokes and explain each one to yourself or a friend.', 'TREND_BASED', 'LOW', true, 'GLOBAL'),
('Chai Craze Check', 'Make or order a masala chai and see why it is a campus favorite.', 'TREND_BASED', 'LOW', true, 'INDIA'),
('Russian Trend Snapshot', 'Find three examples of a current Russian youth trend in music, fashion, or slang.', 'TREND_BASED', 'LOW', true, 'RUSSIA'),
('Micro-Trend Experiment', 'Try one current campus micro-trend for a day, such as a study method, outfit, or snack.', 'TREND_BASED', 'MEDIUM', true, 'GLOBAL'),
('Trend Time Capsule', 'Create a small trend board with five current styles or ideas and save it to revisit later.', 'TREND_BASED', 'MEDIUM', true, 'GLOBAL'),
('Russian Playlist Today', 'Create a playlist of six songs currently popular in Russia and listen on a walk.', 'TREND_BASED', 'MEDIUM', true, 'RUSSIA'),
('Trend Remix Project', 'Create a small project inspired by a current trend, such as a poster, playlist, or outfit.', 'TREND_BASED', 'HIGH', true, 'GLOBAL'),
('Pop-up Trend Night', 'Host or join a mini gathering to try a trend together, like a board-game night or a new snack.', 'TREND_BASED', 'HIGH', true, 'GLOBAL'),
('Campus Dance Step Challenge', 'Learn a short dance step popular on Indian campuses and teach it to a friend or record it for yourself.', 'TREND_BASED', 'HIGH', true, 'INDIA'),
('Library Corner Discovery', 'Find a corner of the library you have never used and spend 45 minutes there studying or relaxing.', 'CAMPUS_LIFE', 'LOW', true, 'GLOBAL'),
('Campus Green Space Pause', 'Visit a quiet green spot on campus and take a 20-minute break there.', 'CAMPUS_LIFE', 'LOW', true, 'GLOBAL'),
('Chai Study Break', 'Invite a classmate for a short chai break between study sessions.', 'CAMPUS_LIFE', 'LOW', true, 'INDIA'),
('Department Tour', 'Visit a department building you have never entered and find one interesting display or lab.', 'CAMPUS_LIFE', 'MEDIUM', true, 'GLOBAL')
ON CONFLICT DO NOTHING;