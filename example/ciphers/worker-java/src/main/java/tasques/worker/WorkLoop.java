package tasques.worker;

import tasques.client.ApiException;
import tasques.client.api.TasksApi;
import tasques.client.models.*;

import java.time.Duration;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Callable;
import java.util.concurrent.atomic.AtomicBoolean;

public class WorkLoop {

    private final String workerId;
    private final TasksApi client;
    private final int claimAmount;
    private final Duration blockFor;
    private final AtomicBoolean stopSignal;
    private final Map<String, TaskHandler> queueTaskHandlers;

    public WorkLoop(final String workerId, final TasksApi client, final int claimAmount, final Duration blockFor, final Map<String, TaskHandler> queueTaskHandlers) {
        this.workerId = workerId;
        this.client = client;
        this.claimAmount = claimAmount;
        this.blockFor = blockFor;
        this.stopSignal = new AtomicBoolean(false);
        this.queueTaskHandlers = queueTaskHandlers;
    }

    public void Run() throws Exception {
        System.out.println("Running");

        final List<String> queueNames = new ArrayList<>(this.queueTaskHandlers.keySet());
        final Callable<List<TaskTask>> makeClaim = () -> {
            final TaskClaim claim = new TaskClaim();
            claim.setAmount(this.claimAmount);
            claim.blockFor("" + this.blockFor.getSeconds() + "s");
            claim.setQueues(queueNames);
            return this.client.claimTasks(this.workerId, claim);
        };

        while (!this.stopSignal.get()) {
            try {
                final List<TaskTask> claimed = makeClaim.call();
                claimed.forEach(t -> {
                    final TaskHandler handler = this.queueTaskHandlers.get(t.getQueue());
                    if (handler == null) {
                        throw new IllegalStateException("No handler for queue [" + t.getQueue() + "]");
                    } else {
                        final TaskHandle handle = new TaskHandle(
                                t, this.client, this.workerId, new AtomicBoolean(false)
                        );
                        if (!handle.isUnclaimed()) {
                            try {
                                try {
                                    TaskSuccess success = handler.handle(handle);
                                    if (success == null) {
                                        this.noResultFailure(t);
                                    } else {
                                        this.markSuccess(t, success);
                                    }
                                } catch (TaskFailedException failed) {
                                    if (failed.f == null) {
                                        this.noResultFailure(t);
                                    } else {
                                        this.markFailed(t, failed.f);
                                    }
                                }
                            } catch (ApiException e) {
                                throw new IllegalStateException("Could not report status on a task", e);
                            }
                        }


                    }
                });
            } catch (ApiException apiErr) {
                // ignore
            }
        }

    }

    private void noResultFailure(final TaskTask t) throws ApiException {
        this.markFailed(t, new TaskFailure().data(new HashMap<String, String>() {{
            put("error", "no result for claimed task");
        }}));
    }

    public void markFailed(final TaskTask t, final TaskFailure f) throws ApiException {
        this.client.markClaimedTaskFailed(t.getQueue(), t.getId(), this.workerId, f);
    }

    public void markSuccess(final TaskTask t, final TaskSuccess s) throws ApiException {
        this.client.markClaimedTaskDone(t.getQueue(), t.getId(), this.workerId, s);
    }


    public static class TaskHandle {
        public TaskTask getTask() {
            return task;
        }

        private final TaskTask task;
        private final TasksApi client;
        private final String workerId;
        private final AtomicBoolean unclaimed;

        public TaskHandle(TaskTask task, TasksApi client, String workerId, AtomicBoolean unclaimed) {
            this.task = task;
            this.client = client;
            this.workerId = workerId;
            this.unclaimed = unclaimed;
        }

        public void unclaim() throws ApiException {
            this.unclaimed.set(true);
            this.client.unclaimExistingTask(this.task.getQueue(), this.task.getId(), this.workerId);
        }

        public void reportIn(final Object data) throws ApiException {
            final TaskNewReport report = new TaskNewReport();
            if (data != null) {
                report.setData(data);
            }
            this.client.reportOnClaimedTask(this.task.getQueue(), this.task.getId(), this.workerId, report);
        }

        public boolean isUnclaimed() {
            return this.unclaimed.get();
        }
    }

    public static class TaskFailedException extends RuntimeException {
        private final TaskFailure f;

        public TaskFailedException(TaskFailure f) {
            this.f = f;
        }
    }

    public interface TaskHandler {
        TaskSuccess handle(final TaskHandle handle) throws TaskFailedException;
    }


}


