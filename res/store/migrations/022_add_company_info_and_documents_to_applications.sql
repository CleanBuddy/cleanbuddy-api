-- Migration 022: Add company info and documents to applications table
-- Extends applications with structured company information and document URLs

ALTER TABLE applications
ADD COLUMN company_info JSONB,
ADD COLUMN documents JSONB,
ADD COLUMN rejection_reason TEXT;

-- Add comments for clarity
COMMENT ON COLUMN applications.company_info IS 'JSON object containing company information: companyName, registrationNumber, taxId, address fields, businessType';
COMMENT ON COLUMN applications.documents IS 'JSON object containing document URLs: identityDocumentUrl, businessRegistrationUrl, insuranceCertificateUrl, additionalDocuments array';
COMMENT ON COLUMN applications.rejection_reason IS 'Reason provided by admin when rejecting the application';

-- Create indexes for JSON queries
CREATE INDEX idx_applications_company_name ON applications USING GIN ((company_info->'companyName'));
CREATE INDEX idx_applications_documents ON applications USING GIN (documents);
