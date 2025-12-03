# Resume Async Upload Workflow Documentation

## Tổng quan

Hệ thống xử lý resume upload một cách bất đồng bộ (asynchronous) sử dụng Kafka làm message queue và WebSocket để gửi real-time updates về trạng thái xử lý cho client.

## Kiến trúc hệ thống

```
┌─────────────┐         ┌──────────────┐         ┌───────────────┐
│   Client    │────────►│  HTTP Server │────────►│     Kafka     │
│             │         │  (Upload API)│         │  (Message Q)  │
└─────────────┘         └──────────────┘         └───────────────┘
       │                        │                         │
       │                        ▼                         │
       │                 ┌──────────────┐                 │
       │                 │   MongoDB    │                 │
       │                 │  (Job Store) │                 │
       │                 └──────────────┘                 │
       │                                                  │
       │                                                  ▼
       │                                          ┌───────────────┐
       │                                          │Resume Worker  │
       │                                          │  (Consumer)   │
       │                                          └───────────────┘
       │                                                  │
       │                                                  ▼
       │                                          ┌───────────────┐
       │                                          │Parser Service │
       │                                          │   (External)  │
       │                                          └───────────────┘
       │                                                  │
       │                                                  ▼
       │                                          ┌───────────────┐
       └─────────────────────────────────────────┤  WebSocket    │
                                                  │     Hub       │
                                                  └───────────────┘
```

## Workflow chi tiết

### Phase 1: Upload Request (Client → Backend)

**Endpoint:** `POST /api/v1/resumes/upload`

**Request:**

```http
POST /api/v1/resumes/upload HTTP/1.1
Host: localhost:8000
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data

file: [PDF file binary data]
```

**Steps:**

1. Client gửi PDF file qua HTTP multipart form-data
2. Backend xác thực JWT token để lấy `user_id`
3. Backend validate:
   - File type phải là PDF
   - File size không vượt quá 10MB
   - User chưa có resume (business rule)

**Response:**

```json
{
  "success": true,
  "message": "Resume upload accepted. Processing will be done asynchronously.",
  "job_id": "65f7a1234567890abcdef123",
  "status": "pending"
}
```

### Phase 2: Job Creation & Kafka Publishing

**Trong backend (upload.go - HandleUploadResumeAsync):**

```go
// 1. Read file data vào memory
fileData, err := io.ReadAll(file)

// 2. Tạo job record trong MongoDB
job := &ResumeParseJob{
    UserID:   userObjectID,
    FileName: "resume.pdf",
    FileData: fileData,  // Lưu tạm file data
    Status:   "pending",
    CreatedAt: time.Now(),
}
jobRepo.Create(ctx, job)

// 3. Gửi message tới Kafka
kafkaMsg := ResumeParseMessage{
    JobID:    job.ID.Hex(),
    UserID:   userID,
    FileName: "resume.pdf",
}
producer.SendMessage(ctx, "resume-parse-jobs", jobID, kafkaMsg)
```

**Kafka Message Format:**

```json
{
  "job_id": "65f7a1234567890abcdef123",
  "user_id": "65f7a1234567890abcdef456",
  "file_name": "resume.pdf"
}
```

**MongoDB Job Document:**

```json
{
  "_id": "65f7a1234567890abcdef123",
  "user_id": "65f7a1234567890abcdef456",
  "file_name": "resume.pdf",
  "file_data": "<binary data>",
  "status": "pending",
  "created_at": "2024-03-15T10:30:00Z",
  "updated_at": "2024-03-15T10:30:00Z"
}
```

### Phase 3: WebSocket Connection (Client ↔ Backend)

**Client establishes WebSocket connection:**

```javascript
// JavaScript example
const token = localStorage.getItem("jwt_token");
const ws = new WebSocket(`ws://localhost:8000/ws?token=${token}`);

ws.onopen = () => {
  console.log("WebSocket connected");
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  if (message.type === "job_status") {
    const { job_id, status, cv_data, resume_id, error_message } =
      message.payload;

    switch (status) {
      case "pending":
        console.log("Job is waiting in queue...");
        break;
      case "processing":
        console.log("Job is being processed...");
        break;
      case "completed":
        console.log("Job completed!", cv_data);
        // Navigate to resume detail page
        window.location.href = `/resumes/${resume_id}`;
        break;
      case "failed":
        console.error("Job failed:", error_message);
        break;
    }
  }
};

ws.onerror = (error) => {
  console.error("WebSocket error:", error);
};

ws.onclose = () => {
  console.log("WebSocket closed");
};
```

**WebSocket Message Format:**

```json
{
  "type": "job_status",
  "payload": {
    "job_id": "65f7a1234567890abcdef123",
    "status": "processing",
    "error_message": "",
    "resume_id": "",
    "cv_data": null
  }
}
```

### Phase 4: Kafka Consumer Processing (Resume Worker)

**Worker Configuration:**

- **Topic:** `resume-parse-jobs`
- **Consumer Group:** `resume-parser-group`
- **Partitions:** 1 (có thể scale horizontal bằng cách tăng partitions)
- **Workers per partition:** 2 (có thể scale vertical bằng cách tăng workers)

**Processing Flow trong resume_worker.go:**

```go
func (w *ResumeWorker) handleMessage(ctx context.Context, msg kafka.Message) error {
    // 1. Parse Kafka message
    var parseMsg ResumeParseMessage
    json.Unmarshal(msg.Value, &parseMsg)

    // 2. Update status to PROCESSING
    jobRepo.UpdateStatus(ctx, parseMsg.JobID, "processing", "", "")

    // 3. Send WebSocket update: PROCESSING
    hub.SendJobStatus(parseMsg.UserID, &JobStatusPayload{
        JobID:  parseMsg.JobID,
        Status: "processing",
    })

    // 4. Get job từ database (để lấy file_data)
    job, _ := jobRepo.GetByID(ctx, parseMsg.JobID)

    // 5. Call Parser Service
    parserResp, err := sendToParserService(ctx, job.FileData, job.FileName)
    if err != nil {
        // Handle failure
        handleJobFailure(ctx, parseMsg.JobID, parseMsg.UserID, err.Error())
        return err
    }

    // 6. Convert và save resume vào user collection
    resume := convertToDataResume(parserResp)
    resumeID, _ := saveResumeToUser(ctx, parseMsg.UserID, resume)

    // 7. Update job status to COMPLETED
    jobRepo.UpdateStatus(ctx, parseMsg.JobID, "completed", "", resumeID)

    // 8. Clear file_data để tiết kiệm storage
    jobRepo.ClearFileData(ctx, parseMsg.JobID)

    // 9. Send WebSocket update: COMPLETED với CV data
    hub.SendJobStatus(parseMsg.UserID, &JobStatusPayload{
        JobID:    parseMsg.JobID,
        Status:   "completed",
        ResumeID: resumeID,
        CVData:   parserResp.CVData,
    })

    return nil
}
```

### Phase 5: Parser Service Call

**Request to External Parser Service:**

```http
POST http://parser-service.com/parse/pdf HTTP/1.1
Content-Type: multipart/form-data

file: [PDF binary data]
```

**Parser Response:**

```json
{
  "success": true,
  "message": "Resume parsed successfully",
  "cv_data": {
    "name": "Nguyễn Văn A",
    "email": "nguyenvana@example.com",
    "phone": "+84 123 456 789",
    "summary": "Experienced Software Engineer...",
    "skills": ["Go", "Python", "Kafka", "MongoDB"],
    "education": [
      {
        "degree": "Bachelor of Computer Science",
        "institution": "UIT - VNU-HCM",
        "graduation_year": 2020,
        "gpa": 3.5
      }
    ],
    "experience": [
      {
        "title": "Senior Backend Developer",
        "company": "Tech Corp",
        "duration": "2020-2024",
        "responsibilities": ["Design microservices", "Implement APIs"],
        "achievements": ["Reduced latency by 50%"]
      }
    ],
    "certifications": ["AWS Certified Developer"],
    "languages": ["Vietnamese", "English"]
  }
}
```

### Phase 6: Save Resume to MongoDB

**Resume Document Structure:**

```json
{
  "_id": "65f7a1234567890abcdef789",
  "resume_detail": {
    "name": "Nguyễn Văn A",
    "email": "nguyenvana@example.com",
    "phone": "+84 123 456 789",
    "summary": "Experienced Software Engineer...",
    "skill": ["Go", "Python", "Kafka", "MongoDB"],
    "education": {
      "degree": "Bachelor of Computer Science",
      "institution": "UIT - VNU-HCM",
      "graduation_year": "2020"
    },
    "experience": {
      "title": "Senior Backend Developer",
      "company": "Tech Corp",
      "duration": "2020-2024",
      "responsibilities": ["Design microservices", "Implement APIs"],
      "achievements": ["Reduced latency by 50%"]
    },
    "certifications": ["AWS Certified Developer"],
    "languages": ["Vietnamese", "English"]
  },
  "version": 1,
  "created_at": "2024-03-15T10:30:05Z"
}
```

**Update User Document:**

```javascript
// MongoDB update operation
db.user.updateOne(
  { _id: ObjectId("65f7a1234567890abcdef456") },
  {
    $push: { resume: <resume_document> },
    $set: { updated_at: new Date() }
  }
)
```

### Phase 7: WebSocket Notification (Complete)

**Final WebSocket Message to Client:**

```json
{
  "type": "job_status",
  "payload": {
    "job_id": "65f7a1234567890abcdef123",
    "status": "completed",
    "resume_id": "65f7a1234567890abcdef789",
    "cv_data": {
      "name": "Nguyễn Văn A",
      "email": "nguyenvana@example.com",
      "phone": "+84 123 456 789",
      "summary": "Experienced Software Engineer...",
      "skills": ["Go", "Python", "Kafka", "MongoDB"],
      "education": [...],
      "experience": [...],
      "certifications": [...],
      "languages": [...]
    }
  }
}
```

## Status Lifecycle

```
┌─────────┐
│ PENDING │ ──► Job được tạo, message đã gửi vào Kafka
└────┬────┘
     │
     ▼
┌────────────┐
│ PROCESSING │ ──► Consumer đang xử lý, gọi Parser Service
└────┬───┬───┘
     │   │
     │   └─────────┐
     ▼             ▼
┌───────────┐  ┌────────┐
│ COMPLETED │  │ FAILED │
└───────────┘  └────────┘
     │             │
     │             └──► Có error_message
     └──► Có resume_id và cv_data
```

## Error Handling

### 1. Upload Phase Errors

- **Invalid token:** 401 Unauthorized
- **Invalid file type:** 400 Bad Request
- **File too large:** 400 Bad Request
- **User already has resume:** 400 Bad Request
- **Kafka send failed:** 500 Internal Server Error
  - Job status updated to `failed` in database

### 2. Processing Phase Errors

- **Parser service unavailable:** Job status → `failed`
- **Parser returned error:** Job status → `failed`
- **Database save error:** Job status → `failed`
- All errors trigger WebSocket notification với `error_message`

### 3. WebSocket Errors

- **Connection lost:** Client tự động reconnect
- **Invalid token:** 401 Unauthorized, connection refused
- **No clients connected:** Message logged nhưng không fail job

## Polling Alternative (Fallback)

Nếu WebSocket không available, client có thể polling job status:

**Endpoint:** `GET /api/v1/resumes/job-status?job_id=xxx`

```javascript
async function pollJobStatus(jobId) {
  const maxAttempts = 60; // 5 minutes (60 * 5s)
  let attempts = 0;

  const poll = async () => {
    const response = await fetch(`/api/v1/resumes/job-status?job_id=${jobId}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await response.json();

    if (data.status === "completed") {
      console.log("Job completed!", data);
      return data;
    } else if (data.status === "failed") {
      throw new Error(data.error_message);
    } else if (attempts < maxAttempts) {
      attempts++;
      setTimeout(poll, 5000); // Poll every 5 seconds
    } else {
      throw new Error("Job timeout");
    }
  };

  return poll();
}
```

## Configuration

### Kafka Configuration (`configs/config.yaml`)

```yaml
kafka:
  brokers:
    - localhost:9092
  username: ${KAFKA_USERNAME}
  password: ${KAFKA_PASSWORD}
  enable_sasl: false
  timeout: 30s
  resume_parse_topic: resume-parse-jobs
  consumer_group: resume-parser-group
  num_partitions: 1 # Scale horizontal: tăng partitions
  num_workers: 2 # Scale vertical: tăng workers per partition
```

### Environment Variables

```bash
# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_USERNAME=
KAFKA_PASSWORD=
KAFKA_ENABLE_SASL=false
KAFKA_RESUME_PARSE_TOPIC=resume-parse-jobs
KAFKA_CONSUMER_GROUP=resume-parser-group

# Database
DATABASE_SOURCE=mongodb://localhost:27017
DATABASE_NAME=jobly

# Services
RESUME_PARSER_URL=http://parser-service:8080

# Auth
JWT_SECRET=your-secret-key
```

## Scaling Strategies

### 1. Horizontal Scaling (Kafka Partitions)

- Tăng `num_partitions` để có nhiều partition
- Mỗi partition có thể được consume bởi một worker khác nhau
- Tốt cho throughput cao

```yaml
kafka:
  num_partitions: 3 # 3 partitions = 3 parallel consumers
  num_workers: 2 # 6 total workers (3 partitions × 2 workers)
```

### 2. Vertical Scaling (Workers per Partition)

- Tăng `num_workers` để có nhiều goroutines xử lý messages từ cùng một partition
- Tốt cho processing time lâu (parsing phức tạp)

```yaml
kafka:
  num_partitions: 1
  num_workers: 5 # 5 concurrent workers trên 1 partition
```

### 3. WebSocket Scaling

- WebSocket Hub sử dụng in-memory map
- Để scale multiple instances, cần implement Redis-based pub/sub
- Hoặc sử dụng sticky sessions

## Monitoring & Observability

### Key Metrics to Track

1. **Job Metrics:**

   - Jobs created per minute
   - Jobs processed per minute
   - Average processing time
   - Success/failure rate

2. **Kafka Metrics:**

   - Consumer lag
   - Messages per second
   - Partition distribution

3. **WebSocket Metrics:**
   - Active connections
   - Messages sent per second
   - Connection errors

### Logging

```go
// Tất cả các stage đều có logs:
log.Infof("Resume parse job created: %s", jobID)
log.Infof("Processing resume parse job: %s", jobID)
log.Infof("Resume parse job completed: %s, resume ID: %s", jobID, resumeID)
log.Errorf("Failed to parse resume: %v", err)
```

## Testing

### 1. Unit Tests

```go
// Test job creation
func TestCreateResumeParseJob(t *testing.T) { ... }

// Test Kafka message handling
func TestHandleKafkaMessage(t *testing.T) { ... }

// Test WebSocket broadcasting
func TestSendJobStatus(t *testing.T) { ... }
```

### 2. Integration Tests

```bash
# Start Kafka, MongoDB, Parser Service
docker-compose up -d

# Run integration tests
go test -tags=integration ./...
```

### 3. Load Testing

```bash
# Sử dụng k6 hoặc similar tool
k6 run --vus 100 --duration 30s load-test.js
```

## Troubleshooting

### Problem: Jobs stuck in "pending"

**Cause:** Kafka consumer không chạy hoặc crash  
**Solution:** Check logs, restart worker

### Problem: WebSocket không nhận updates

**Cause:** Connection lost hoặc Hub không broadcast  
**Solution:** Client reconnect, check Hub logs

### Problem: Parser service timeout

**Cause:** File quá lớn hoặc phức tạp  
**Solution:** Tăng timeout, optimize parser

### Problem: High memory usage

**Cause:** File data được lưu trong MongoDB  
**Solution:** ClearFileData được gọi sau processing

## Best Practices

1. **Always use WebSocket for real-time updates**
2. **Keep job records for audit trail**
3. **Clear file_data after processing to save storage**
4. **Set appropriate timeouts for parser service**
5. **Monitor Kafka consumer lag**
6. **Implement retry logic for transient failures**
7. **Use structured logging for easier debugging**
8. **Add circuit breaker for parser service calls**

## Future Improvements

1. **Retry mechanism:** Tự động retry failed jobs
2. **Priority queue:** VIP users có priority cao hơn
3. **Batch processing:** Xử lý multiple files cùng lúc
4. **Caching:** Cache parsed results
5. **Analytics:** Track parsing accuracy, common errors
6. **Rate limiting:** Giới hạn số uploads per user per day
7. **File storage:** Move file data to S3/MinIO thay vì MongoDB
8. **Dead letter queue:** Move failed messages to DLQ for manual review

## Conclusion

Hệ thống async upload resume giúp:

- ✅ Tăng responsiveness cho user (không phải chờ parsing)
- ✅ Scale tốt hơn (horizontal scaling via Kafka partitions)
- ✅ Resilient (failed jobs không ảnh hưởng đến system)
- ✅ Real-time feedback (WebSocket updates)
- ✅ Audit trail (job history trong database)
