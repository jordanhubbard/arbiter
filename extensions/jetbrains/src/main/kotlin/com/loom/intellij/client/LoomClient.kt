package com.loom.intellij.client

import com.google.gson.Gson
import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.util.concurrent.TimeUnit

data class Message(
    val role: String,
    val content: String
)

data class ChatRequest(
    val messages: List<Message>,
    val model: String = "default",
    val temperature: Double = 0.7,
    val max_tokens: Int = 2000
)

data class Choice(
    val message: Message,
    val finish_reason: String
)

data class ChatResponse(
    val id: String,
    val choices: List<Choice>
)

class LoomClient(private var apiEndpoint: String, private var apiKey: String) {
    private val gson = Gson()
    private val client = OkHttpClient.Builder()
        .connectTimeout(60, TimeUnit.SECONDS)
        .readTimeout(60, TimeUnit.SECONDS)
        .writeTimeout(60, TimeUnit.SECONDS)
        .build()

    fun updateConfig(endpoint: String, key: String) {
        this.apiEndpoint = endpoint
        this.apiKey = key
    }

    fun sendMessage(messages: List<Message>): ChatResponse {
        val request = ChatRequest(messages = messages)
        val json = gson.toJson(request)

        val requestBody = json.toRequestBody("application/json".toMediaType())
        val httpRequest = Request.Builder()
            .url("$apiEndpoint/api/v1/chat/completions")
            .post(requestBody)
            .apply {
                if (apiKey.isNotEmpty()) {
                    header("Authorization", "Bearer $apiKey")
                }
            }
            .build()

        client.newCall(httpRequest).execute().use { response ->
            if (!response.isSuccessful) {
                throw IOException("API request failed: ${response.code} ${response.message}")
            }

            val responseBody = response.body?.string()
                ?: throw IOException("Empty response body")

            return gson.fromJson(responseBody, ChatResponse::class.java)
        }
    }

    fun healthCheck(): Boolean {
        return try {
            val request = Request.Builder()
                .url("$apiEndpoint/health")
                .get()
                .build()

            client.newCall(request).execute().use { response ->
                response.isSuccessful
            }
        } catch (e: Exception) {
            false
        }
    }
}
