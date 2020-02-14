use crate::messages::{MessagePartialUpdate, MessagesRepo};

use async_trait::async_trait;
use base64;
use md5;
use serde::Deserialize;
use serde_json::json;
use tasques_client::models::{TaskFailure, TaskNewReport, TaskSuccess};
use tasques_worker::work_loop::{TaskHandle, TaskHandler};

static LOWER: [char; 26] = [
    'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's',
    't', 'u', 'v', 'w', 'x', 'y', 'z',
];
static UPPER: [char; 26] = [
    'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S',
    'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
];

pub struct CipherHandler {
    pub messages_repo: MessagesRepo,
}

#[async_trait]
impl TaskHandler for CipherHandler {
    async fn handle(&self, handle: TaskHandle) -> Result<TaskSuccess, TaskFailure> {
        let report_in = handle
            .report_in(TaskNewReport {
                data: Some(json!({
                    "status": format!("starting on task: {}", handle.task.id)
                })),
            })
            .await;
        match report_in {
            Ok(_) => {
                let kind = handle.task.kind.clone();
                if let Some(args) = handle.task.args {
                    match serde_json::from_value::<MessageCipherTask>(args) {
                        Ok(task) => {
                            let maybe_retrieved_msg =
                                self.messages_repo.get(&task.message_id).await;
                            match maybe_retrieved_msg {
                                Ok(msg) => {
                                    let encoded_or_err = match kind.as_str() {
                                        "Rot1" => {
                                            Ok(MessagePartialUpdate::rot_1(rotate(&msg.plain, 1)))
                                        }

                                        "Rot13" => {
                                            Ok(MessagePartialUpdate::rot_13(rotate(&msg.plain, 13)))
                                        }
                                        "Base64" => Ok(MessagePartialUpdate::base_64(to_base_64(
                                            &msg.plain,
                                        ))),
                                        "Md5" => {
                                            Ok(MessagePartialUpdate::md_5(to_md_5(&msg.plain)))
                                        }
                                        _ => Err(TaskFailure {
                                            data: Some(json!({
                                                "err":
                                                    format!("unsupported kind {}", handle.task.kind)
                                            })),
                                        }),
                                    };
                                    match encoded_or_err {
                                        Ok(partial) => self
                                            .messages_repo
                                            .update(&task.message_id, partial)
                                            .await
                                            .map(|_| TaskSuccess {
                                                data: Some(
                                                    json!({ "success": "Done and done by Rust"}),
                                                ),
                                            })
                                            .map_err(|e| TaskFailure {
                                                data: Some(json!({ "err": format!("{}", e) })),
                                            }),
                                        Err(e) => Err(e),
                                    }
                                }
                                Err(e) => Err(TaskFailure {
                                    data: Some(json!({ "err": format!("{}", e) })),
                                }),
                            }
                        }
                        Err(serdes_err) => Err(TaskFailure {
                            data: Some(json!({ "err": format!("{}", serdes_err) })),
                        }),
                    }
                } else {
                    Err(TaskFailure {
                        data: Some(json!({
                            "err": "No arguments"
                        })),
                    })
                }
            }
            Err(e) => Err(TaskFailure {
                data: Some(json!({
                    "err": format!("failed to report in due to {}", e)
                })),
            }),
        }
    }
}

fn rotate(s: &str, shift: usize) -> String {
    let mut chars = s.chars().collect::<Vec<char>>();
    for i in 0..chars.len() {
        let c = chars[i];
        let maybe_loopk_up = if c >= UPPER[0] && c <= UPPER[UPPER.len() - 1] {
            Some(UPPER)
        } else if c >= LOWER[0] && c <= LOWER[LOWER.len() - 1] {
            Some(LOWER)
        } else {
            None
        };
        if let Some(look_up) = maybe_loopk_up {
            let offset = c as usize - look_up[0] as usize;
            let mut shifted_idx = offset + shift;
            if shifted_idx > look_up.len() - 1 {
                shifted_idx = shifted_idx % look_up.len()
            }
            chars[i] = look_up[shifted_idx];
        }
    }
    chars.iter().collect()
}

fn to_base_64(s: &str) -> String {
    base64::encode(s)
}

fn to_md_5(s: &str) -> String {
    format!("{:x}", md5::compute(s))
}

#[derive(Deserialize)]
struct MessageCipherTask {
    message_id: String,
}
