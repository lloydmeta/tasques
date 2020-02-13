## Tasques

Task queues backed by ES: Tasques.

### Features:

Some of these may be goals :p

- Easily scalable:
  - Servers are stateless; easily spin more up as needed
  - The storage engine Elasticsearch. That says it all
- Tasks can be configured
  - Priority
  - When to run
  - Retries
    - Tasks can be configured to retry X times, with exponential increase in run times
- Timeouts
  - Tasks that are picked up by workers that either don't report in or finish on time get timed out.
- Unclaiming
  - Tasks that were picked up but can't be handled now can be requeued without consequence.
