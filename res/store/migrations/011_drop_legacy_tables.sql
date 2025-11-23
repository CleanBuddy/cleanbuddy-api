-- Migration 011: Drop legacy tables (teams, projects, tasks, invitation_codes)
-- This removes all team/project functionality and related systems

-- Drop tables in correct order (respecting foreign key dependencies)

-- Drop tasks table (queue system - no longer needed)
DROP TABLE IF EXISTS tasks CASCADE;

-- Drop projects table (depends on teams)
DROP TABLE IF EXISTS projects CASCADE;

-- Drop team-related junction tables
DROP TABLE IF EXISTS team_member_invites CASCADE;
DROP TABLE IF EXISTS team_members CASCADE;

-- Drop teams table
DROP TABLE IF EXISTS teams CASCADE;

-- Drop invitation codes table (replaced by application system)
DROP TABLE IF EXISTS invitation_codes CASCADE;
