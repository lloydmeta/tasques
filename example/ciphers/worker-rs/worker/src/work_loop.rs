use crate::config::{TasquesWorker, WorkerId};
use async_trait::async_trait;
use log::*;
use std::collections::HashMap;
use std::error::Error as StdError;
use std::fmt::{Debug, Display, Error, Formatter};
use std::io;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time;
use tasques_client::apis::{self, TasksApi, TasksApiClient};
use tasques_client::models::{TaskClaim, TaskFailure, TaskNewReport, TaskSuccess, TaskTask};
use tokio::{select, signal, task};

#[async_trait]
pub trait TaskHandler {
    async fn handle(&self, handle: TaskHandle) -> Result<TaskSuccess, TaskFailure>;
}

pub struct TaskHandle {
    client: Arc<TasksApiClient>,
    pub task: TaskTask,
    worker_id: WorkerId,

    unclaimed: AtomicBool,
}

impl TaskHandle {
    pub fn is_unclaimed(&self) -> bool {
        self.unclaimed.load(Ordering::Relaxed)
    }

    pub async fn unclaim(&self) -> Result<(), WorkLoopErr> {
        let client_clone = self.client.clone();
        let queue = self.task.queue.clone();
        let task_id = self.task.id.clone();
        let worker_id = self.worker_id.clone();
        let t = task::spawn_blocking(move || {
            client_clone.unclaim_existing_task(
                queue.as_str(),
                task_id.as_str(),
                worker_id.0.as_str(),
            )
        });
        t.await??;
        self.unclaimed.store(true, Ordering::Relaxed);
        Ok(())
    }

    pub async fn report_in(&self, report: TaskNewReport) -> Result<TaskTask, WorkLoopErr> {
        let client_clone = self.client.clone();
        let queue = self.task.queue.clone();
        let task_id = self.task.id.clone();
        let worker_id = self.worker_id.clone();
        let t = task::spawn_blocking(move || {
            client_clone.report_on_claimed_task(
                queue.as_str(),
                task_id.as_str(),
                worker_id.0.as_str(),
                report,
            )
        });
        let updated_task = t.await??;
        Ok(updated_task)
    }
}

pub struct QueuesToHandler(pub HashMap<String, Box<dyn TaskHandler>>);

/// This is the basic work loop.
///
/// It tries to wrap the blocking Tasques API client using Tokio primitives.
pub struct WorkLoop {
    pub worker_id: WorkerId,
    pub client: Arc<TasksApiClient>,
    pub queues_to_handlers: QueuesToHandler,
    pub claim_amount: u32,
    pub block_for: time::Duration,

    stopped: Arc<AtomicBool>,
}

impl WorkLoop {
    pub fn build(worker_config: TasquesWorker, queues_to_handlers: QueuesToHandler) -> WorkLoop {
        WorkLoop {
            worker_id: worker_config.id,
            client: Arc::new(worker_config.server.into_client()),
            queues_to_handlers,
            claim_amount: worker_config.claim_amount,
            block_for: worker_config.block_for,
            stopped: Arc::new(AtomicBool::new(false)),
        }
    }

    fn build_claim_req(&self) -> TaskClaim {
        let queue_names = self
            .queues_to_handlers
            .0
            .keys()
            .map(|k| k.to_owned())
            .collect::<Vec<String>>();

        let mut claim = TaskClaim::new(queue_names);
        claim.amount = Some(self.claim_amount as i32);
        claim.block_for = Some(format!("{}s", self.block_for.as_secs()));
        claim
    }

    /// Starts a work loop that tries to exit gracefully when hit with SIGINT or SIGTERM
    pub async fn run(self) -> Result<(), WorkLoopErr> {
        let stop_signaller = self.stopped.clone();
        let mut sigint = signal::unix::signal(signal::unix::SignalKind::interrupt())?;
        let mut sigterm = signal::unix::signal(signal::unix::SignalKind::terminate())?;
        task::spawn(async move {
            select! {
               _ = sigint.recv() => {
                   info!("SIGINT received");
               }
               _ = sigterm.recv() => {
                   info!("SIGTERM received");
               }
            }
            stop_signaller.store(true, Ordering::Relaxed);
        });
        while !self.stopped.load(Ordering::Relaxed) {
            self.run_once().await?;
        }
        info!("Exiting loop gracefully");
        Ok(())
    }

    async fn run_once(&self) -> Result<(), WorkLoopErr> {
        let claimed_tasks = self.claim_tasks().await?;
        for t in claimed_tasks.into_iter() {
            let task_queue = t.queue.clone();
            let task_id = t.id.clone();
            let maybe_handler = self.queues_to_handlers.0.get(t.queue.as_str());
            match maybe_handler {
                Some(handler) => {
                    let handle = TaskHandle {
                        client: self.client.clone(),
                        task: t,
                        worker_id: self.worker_id.clone(),
                        unclaimed: AtomicBool::new(false),
                    };
                    let handling_result = handler.handle(handle).await;
                    match handling_result {
                        Ok(s) => self.mark_done(&task_queue, &task_id, s).await?,
                        Err(f) => self.mark_failed(&task_queue, &task_id, f).await?,
                    }
                }
                None => {
                    return Err(WorkLoopErr::NoHandlerForQueue(t.queue));
                }
            }
        }
        Ok(())
    }

    async fn mark_done(
        &self,
        queue: &str,
        id: &str,
        success: TaskSuccess,
    ) -> Result<(), WorkLoopErr> {
        let client_clone = self.client.clone();
        let worker_id = self.worker_id.clone();
        let queue_string = queue.to_owned();
        let id_string = id.to_owned();
        let t = task::spawn_blocking(move || {
            client_clone.mark_claimed_task_done(&queue_string, &id_string, &worker_id.0, success)
        });
        t.await??;
        Ok(())
    }

    async fn mark_failed(
        &self,
        queue: &str,
        id: &str,
        failure: TaskFailure,
    ) -> Result<(), WorkLoopErr> {
        let client_clone = self.client.clone();
        let worker_id = self.worker_id.clone();
        let queue_string = queue.to_owned();
        let id_string = id.to_owned();
        let t = task::spawn_blocking(move || {
            client_clone.mark_claimed_task_failed(&queue_string, &id_string, &worker_id.0, failure)
        });
        t.await??;
        Ok(())
    }

    async fn claim_tasks(&self) -> Result<Vec<TaskTask>, WorkLoopErr> {
        let claim_req = self.build_claim_req();
        let client_clone = self.client.clone();
        let worker_id = self.worker_id.clone();
        let t = task::spawn_blocking(move || client_clone.claim_tasks(&worker_id.0, claim_req));
        let r = t.await??;
        Ok(r)
    }
}

// Errors

pub enum WorkLoopErr {
    ApiError(apis::Error),
    IoError(io::Error),
    NoHandlerForQueue(String),
    RuntimeError(task::JoinError),
}

impl Debug for WorkLoopErr {
    fn fmt(&self, f: &mut Formatter<'_>) -> Result<(), Error> {
        match self {
            WorkLoopErr::ApiError(inner) => {
                f.write_fmt(format_args!("WorkLoopErr, API err [{:?}]", inner))
            }
            WorkLoopErr::IoError(io) => f.write_fmt(format_args!("WorkLoopErr, IO err [{:?}]", io)),
            WorkLoopErr::NoHandlerForQueue(q) => {
                f.write_fmt(format_args!("WorkLoopErr, no handler for queue [{}]", q))
            }
            WorkLoopErr::RuntimeError(q) => {
                f.write_fmt(format_args!("WorkLoopErr, no handler for queue [{}]", q))
            }
        }
    }
}

impl Display for WorkLoopErr {
    fn fmt(&self, f: &mut Formatter<'_>) -> Result<(), Error> {
        f.write_fmt(format_args!("{:?}", self)) // delegate to debug
    }
}

impl StdError for WorkLoopErr {}

impl From<apis::Error> for WorkLoopErr {
    fn from(e: apis::Error) -> Self {
        WorkLoopErr::ApiError(e)
    }
}

impl From<task::JoinError> for WorkLoopErr {
    fn from(e: task::JoinError) -> Self {
        WorkLoopErr::RuntimeError(e)
    }
}

impl From<io::Error> for WorkLoopErr {
    fn from(e: io::Error) -> Self {
        WorkLoopErr::IoError(e)
    }
}
