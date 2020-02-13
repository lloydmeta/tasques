package tasques.ciphers;

import tasques.client.models.TaskFailure;
import tasques.client.models.TaskSuccess;
import tasques.messages.Message;
import tasques.worker.WorkLoop;

import javax.xml.bind.DatatypeConverter;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Base64;
import java.util.HashMap;
import java.util.Map;

public class Handler implements WorkLoop.TaskHandler {

    private final Message.Repo repo;

    public Handler(Message.Repo repo) {
        this.repo = repo;
    }

    @Override
    public TaskSuccess handle(WorkLoop.TaskHandle handle) throws WorkLoop.TaskFailedException {
        try {
            final String msgId = ((Map<String, String>) handle.getTask().getArgs()).get("message_id");
            final Message msg = this.repo.get(msgId);
            final String kind = handle.getTask().getKind();
            String encoded;
            switch (kind) {
                case "Rot1":
                    encoded = rotate(msg.getPlain(), 1);
                    this.repo.updateRot1(msg.getId(), encoded);
                    break;
                case "Rot13":
                    encoded = rotate(msg.getPlain(), 13);
                    this.repo.updateRot13(msg.getId(), encoded);

                    break;
                case "Base64":
                    encoded = base64Encode(msg.getPlain());
                    this.repo.updateBase64(msg.getId(), encoded);

                    break;
                case "Md5":
                    encoded = md5Hash(msg.getPlain());
                    this.repo.updateMd5(msg.getId(), encoded);
                    break;
                default:
                    throw new IllegalArgumentException("Cipher does not support: [" + kind + "]");

            }
            return new TaskSuccess().data(new HashMap<String, String>() {{
                put("java-success", encoded);
            }});
        } catch (Exception e) {
            throw new WorkLoop.TaskFailedException(new TaskFailure().data(new HashMap<String, String>() {{
                put("err", e.getMessage());
            }}));
        }


    }

    private static final String lower = "abcdefghijklmnopqrstuvwxyz";
    private static final char[] lowerArray = lower.toCharArray();
    private static final String upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    private static final char[] upperArray = upper.toCharArray();


    public static String rotate(String s, int shift) {
        final char[] chars = s.toCharArray();
        for (int i = 0; i < chars.length; i++) {
            final char c = chars[i];
            char[] lookup = null;
            if (c >= upperArray[0] && c <= upperArray[upperArray.length - 1]) {
                lookup = upperArray;
            } else if (c >= lowerArray[0] && c <= lowerArray[lowerArray.length - 1]) {
                lookup = lowerArray;
            }
            if (lookup != null) {
                int offset = c - lookup[0];
                int shiftedIdx = offset + shift;
                if (shiftedIdx > lookup.length - 1) {
                    shiftedIdx = shiftedIdx % lookup.length;
                }
                chars[i] = lookup[shiftedIdx];
            }
        }
        return new String(chars);
    }

    public static String base64Encode(final String s) {
        return Base64.getEncoder().encodeToString(s.getBytes());
    }

    public static String md5Hash(final String s) throws NoSuchAlgorithmException {
        MessageDigest md = MessageDigest.getInstance("MD5");
        md.update(s.getBytes());
        byte[] digest = md.digest();
        return DatatypeConverter
                .printHexBinary(digest).toUpperCase();
    }
}
