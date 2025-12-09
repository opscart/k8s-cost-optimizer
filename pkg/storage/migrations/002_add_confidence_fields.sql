-- Migration 002: Add confidence and pattern analysis fields
-- Week 9: Pattern-based recommendations

-- Add confidence columns to recommendations table
ALTER TABLE recommendations 
ADD COLUMN IF NOT EXISTS confidence VARCHAR(10),
ADD COLUMN IF NOT EXISTS data_quality DECIMAL(3,2),
ADD COLUMN IF NOT EXISTS pattern_info VARCHAR(255),
ADD COLUMN IF NOT EXISTS has_sufficient_data BOOLEAN DEFAULT false;

-- Add index for confidence filtering
CREATE INDEX IF NOT EXISTS idx_recommendations_confidence ON recommendations(confidence);

-- Update schema version
INSERT INTO schema_version (version) VALUES (2) ON CONFLICT (version) DO NOTHING;