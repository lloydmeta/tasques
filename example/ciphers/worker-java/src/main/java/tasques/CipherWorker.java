package tasques;

import com.typesafe.config.Config;
import org.apache.http.HttpHost;
import org.apache.http.auth.AuthScope;
import org.apache.http.auth.UsernamePasswordCredentials;
import org.apache.http.client.CredentialsProvider;
import org.apache.http.impl.client.BasicCredentialsProvider;
import org.elasticsearch.client.RestClient;
import org.elasticsearch.client.RestClientBuilder;
import org.elasticsearch.client.RestHighLevelClient;
import tasques.ciphers.Handler;
import tasques.client.ApiClient;
import tasques.client.api.TasksApi;
import tasques.messages.Message;
import tasques.worker.WorkLoop;
import com.typesafe.config.ConfigFactory;

import java.util.HashMap;

public class CipherWorker {
    public static void main(String[] args) {

        Config config = ConfigFactory.load();

        HttpHost[] esAddresses = config.
                getStringList("elasticsearch.addresses").
                stream().
                map(HttpHost::create).
                toArray(HttpHost[]::new);

        RestClientBuilder restClientBuilder = RestClient.builder(esAddresses);
        if (config.hasPath("elasticsearch.user")) {
            final CredentialsProvider credentialsProvider = new BasicCredentialsProvider();
            credentialsProvider.setCredentials(
                    AuthScope.ANY,
                    new UsernamePasswordCredentials(
                            config.getString("elasticsearch.user.name"),
                            config.getString("elasticsearch.user.password")
                    )
            );
            restClientBuilder.setHttpClientConfigCallback(builder -> builder.setDefaultCredentialsProvider(credentialsProvider));
        }

        RestHighLevelClient client = new RestHighLevelClient(restClientBuilder);
        Message.Repo messageRepo = new Message.Repo(client);
        Handler handler = new Handler(messageRepo);
        HashMap<String, WorkLoop.TaskHandler> queuesToHandler = new HashMap<String, WorkLoop.TaskHandler>() {{
            put(config.getString("cipher_worker.queue"), handler);
        }};

        TasksApi apiClient = new TasksApi(new ApiClient().setBasePath("http://localhost:8080"));

        Config workerConfig = config.getConfig("cipher_worker.tasques_worker");
        WorkLoop workLoop = new WorkLoop(
                workerConfig.getString("id"),
                apiClient,
                workerConfig.getInt("claim_amount"),
                workerConfig.getDuration("block_for"),
                queuesToHandler
        );

        try {
            workLoop.Run();
        } catch (Exception e) {
            e.printStackTrace();
        }

    }
}
