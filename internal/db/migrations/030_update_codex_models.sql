-- Migration 030: Move Codex agents off retired models
UPDATE agents SET model = 'gpt-6-luna' WHERE runner = 'codex' AND model = 'gpt-5.4-instant';
UPDATE agents SET model = 'gpt-6.1-sol' WHERE runner = 'codex' AND model NOT IN ('', 'default', 'gpt-6.1-sol', 'gpt-6-astra', 'gpt-6.1-terra', 'gpt-6-luna');
