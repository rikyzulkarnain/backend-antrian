-- Foreign-key indexes on queues. Analytics/report queries join and group by
-- staff (user_id) and counter (counter_id); without these the planner does a
-- sequential scan of queues per join. Low write cost, meaningful read speedup
-- on the dashboard and reports.
CREATE INDEX IF NOT EXISTS idx_queues_user_id    ON queues(user_id);
CREATE INDEX IF NOT EXISTS idx_queues_counter_id ON queues(counter_id);
