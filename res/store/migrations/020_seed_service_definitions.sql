-- Migration 020: Seed initial service definitions and add-ons
-- Insert default service types and add-ons for the platform

-- Insert service definitions
INSERT INTO service_definitions (id, type, name, description, base_hours, price_multiplier, is_active) VALUES
    ('svc_general', 'general', 'General Cleaning', 'Standard cleaning service including dusting, vacuuming, mopping, and bathroom cleaning', 3.0, 1.0, true),
    ('svc_deep', 'deep', 'Deep Cleaning', 'Thorough deep cleaning including all standard tasks plus detailed cleaning of hard-to-reach areas, inside appliances, and intensive scrubbing', 5.0, 1.5, true),
    ('svc_move', 'move_in_out', 'Move In/Out Cleaning', 'Complete cleaning for moving in or out of a property, ensuring the space is spotless for new occupants', 4.0, 1.3, true)
ON CONFLICT (type) DO NOTHING;

-- Insert add-on definitions
INSERT INTO service_add_on_definitions (id, add_on, name, description, fixed_price, estimated_hours, is_active) VALUES
    ('addon_oven', 'oven', 'Oven Cleaning', 'Deep cleaning of oven interior, racks, and glass door', 3000, 0.5, true), -- 30 RON
    ('addon_windows', 'windows', 'Window Cleaning', 'Interior and exterior window cleaning (up to 10 windows)', 2500, 1.0, true), -- 25 RON
    ('addon_fridge', 'fridge', 'Refrigerator Cleaning', 'Deep cleaning of refrigerator interior, shelves, and drawers', 2000, 0.5, true), -- 20 RON
    ('addon_garage', 'garage', 'Garage Cleaning', 'Sweeping, organizing, and cleaning of garage space', 4000, 1.5, true) -- 40 RON
ON CONFLICT (add_on) DO NOTHING;
