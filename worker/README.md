## Worker SDK

This provides a basic workloop that:

1. Gracefully exits without losing tasks (with a configurable timeout)
2. Handles API errors
3. Groups Task claims from multiple Queues in a single request for every run of the loop