-- Migration 017: Create reviews table
-- Customer reviews for cleaners after bookings

CREATE TABLE IF NOT EXISTS reviews (
    id VARCHAR(50) PRIMARY KEY,
    booking_id VARCHAR(50) NOT NULL UNIQUE,
    customer_id VARCHAR(50) NOT NULL,
    cleaner_id VARCHAR(50) NOT NULL,
    cleaner_profile_id VARCHAR(50) NOT NULL,

    -- Rating (1-5 stars)
    rating INTEGER NOT NULL,

    -- Review Content
    title VARCHAR(200),
    comment TEXT,

    -- Detailed Ratings (optional breakdown)
    quality_rating INTEGER,
    punctuality_rating INTEGER,
    professionalism_rating INTEGER,
    value_rating INTEGER,

    -- Moderation
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'flagged'
    flag_reason TEXT,
    moderation_note TEXT,
    moderated_by_id VARCHAR(50),
    moderated_at TIMESTAMP,

    -- Response from Cleaner (optional feature)
    cleaner_response TEXT,
    responded_at TIMESTAMP,

    -- Helpfulness tracking
    helpful_count INTEGER NOT NULL DEFAULT 0,
    not_helpful_count INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_reviews_booking FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_reviews_customer FOREIGN KEY (customer_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_reviews_cleaner FOREIGN KEY (cleaner_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_reviews_cleaner_profile FOREIGN KEY (cleaner_profile_id) REFERENCES cleaner_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_reviews_moderated_by FOREIGN KEY (moderated_by_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT check_rating CHECK (rating >= 1 AND rating <= 5),
    CONSTRAINT check_quality_rating CHECK (quality_rating IS NULL OR (quality_rating >= 1 AND quality_rating <= 5)),
    CONSTRAINT check_punctuality_rating CHECK (punctuality_rating IS NULL OR (punctuality_rating >= 1 AND punctuality_rating <= 5)),
    CONSTRAINT check_professionalism_rating CHECK (professionalism_rating IS NULL OR (professionalism_rating >= 1 AND professionalism_rating <= 5)),
    CONSTRAINT check_value_rating CHECK (value_rating IS NULL OR (value_rating >= 1 AND value_rating <= 5)),
    CONSTRAINT check_status CHECK (status IN ('pending', 'approved', 'rejected', 'flagged'))
);

-- Indexes for efficient queries
CREATE INDEX idx_review_booking ON reviews(booking_id);
CREATE INDEX idx_review_customer ON reviews(customer_id);
CREATE INDEX idx_review_cleaner ON reviews(cleaner_id);
CREATE INDEX idx_review_cleaner_profile ON reviews(cleaner_profile_id);
CREATE INDEX idx_review_status ON reviews(status);
CREATE INDEX idx_review_created ON reviews(created_at);
CREATE INDEX idx_review_rating ON reviews(rating);

-- Composite index for approved reviews by cleaner
CREATE INDEX idx_review_cleaner_approved ON reviews(cleaner_profile_id, status) WHERE status = 'approved';
