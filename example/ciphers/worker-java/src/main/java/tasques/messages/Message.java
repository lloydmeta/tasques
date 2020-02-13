package tasques.messages;

import org.elasticsearch.action.get.GetRequest;
import org.elasticsearch.action.update.UpdateRequest;
import org.elasticsearch.client.RequestOptions;
import org.elasticsearch.client.RestHighLevelClient;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

public class Message {

    private final String id;
    private final String plain;

    public Message(String id, String plain) {
        this.id = id;
        this.plain = plain;
    }

    public String getPlain() {
        return plain;
    }

    public String getId() {
        return id;
    }

    public static class Repo {

        private final static String messagesIndex = "messages_and_their_ciphers";

        private final RestHighLevelClient esClient;

        public Repo(final RestHighLevelClient esClient) {
            this.esClient = esClient;
        }

        public Message get(final String id) throws IOException {
            final GetRequest req = new GetRequest(messagesIndex, id);
            final Map<String, Object> map = this.esClient.get(req, RequestOptions.DEFAULT).getSource();
            if (map == null) {
                return null;
            } else {
                return new Message(id, (String) map.get("plain"));
            }
        }

        public void updateRot1(String id, String value) throws IOException {
            this.updateField(id, "rot_1", value);
        }

        public void updateRot13(String id, String value) throws IOException {
            this.updateField(id, "rot_13", value);
        }

        public void updateBase64(String id, String value) throws IOException {
            this.updateField(id, "base_64", value);
        }

        public void updateMd5(String id, String value) throws IOException {
            this.updateField(id, "md_5", value);
        }


        private void updateField(final String id, final String field, final String value) throws IOException {
            try {
                final UpdateRequest req = new UpdateRequest(messagesIndex, id);
                Map<String, Object> map = new HashMap<String, Object>() {{
                    put(field, value);
                }};
                req.doc(map);
                req.retryOnConflict(1000);
                this.esClient.update(req, RequestOptions.DEFAULT);

            } catch (Exception e) {
                e.printStackTrace();
                throw e;
            }
        }
    }

}
