# API Endpoints - Pick JD for Evaluation

Tài liệu mô tả 2 API endpoints mới cho tính năng user chọn JD để evaluate CV.

---

## 1. Get Jobs For Evaluation

Lấy danh sách các jobs mà user có thể chọn để evaluate CV.

### Endpoint

```
GET /api/v1/evaluation/jobs
```

### Authentication

- **Required**: JWT Token (Bearer)

### Request

Không có request body.

### Response

```json
{
  "viewed_jobs": [
    {
      "id": "675a1b2c3d4e5f6789012345",
      "title": "Senior Backend Developer",
      "company_name": "ABC Company",
      "location": "Ho Chi Minh City",
      "time_on_sight": 120
    }
  ],
  "saved_jobs": [
    {
      "id": "675a1b2c3d4e5f6789012346",
      "title": "Frontend Developer",
      "company_name": "XYZ Corp",
      "location": "Ha Noi",
      "time_on_sight": 0
    }
  ]
}
```

### Response Fields

| Field         | Type  | Description                                                                                                   |
| ------------- | ----- | ------------------------------------------------------------------------------------------------------------- |
| `viewed_jobs` | array | Danh sách jobs user đã xem (từ tracking time_on_sight), giới hạn 10 jobs, sắp xếp theo thời gian xem giảm dần |
| `saved_jobs`  | array | Danh sách jobs user đã lưu (từ field `saved_jd_id` trong collection user)                                     |

### JobForEvaluation Object

| Field           | Type   | Description                                                                                   |
| --------------- | ------ | --------------------------------------------------------------------------------------------- |
| `id`            | string | Job ID (MongoDB ObjectID)                                                                     |
| `title`         | string | Tiêu đề công việc                                                                             |
| `company_name`  | string | Tên công ty                                                                                   |
| `location`      | string | Địa điểm làm việc                                                                             |
| `time_on_sight` | int32  | Thời gian user đã xem job (giây). Chỉ có giá trị > 0 với `viewed_jobs`, `saved_jobs` luôn = 0 |

---

## 2. Evaluate With JD

Evaluate CV với một JD cụ thể do user chọn. API này chạy **đồng bộ** (không qua message queue).

### Endpoint

```
POST /api/v1/evaluation/evaluate-with-jd
```

### Authentication

- **Required**: JWT Token (Bearer)

### Request Body

```json
{
  "resume_id": "675a1b2c3d4e5f6789012347",
  "job_id": "675a1b2c3d4e5f6789012345"
}
```

| Field       | Type   | Required | Description                      |
| ----------- | ------ | -------- | -------------------------------- |
| `resume_id` | string | Yes      | ID của resume cần evaluate       |
| `job_id`    | string | Yes      | ID của job được chọn để evaluate |

### Response

```json
{
  "evaluation": {
    "cv_name": "Nguyen Van A",
    "overall_score": 78.5,
    "grade": "B+",
    "score_breakdown": {
      "skills_score": 80.0,
      "experience_score": 75.0,
      "education_score": 85.0,
      "completeness_score": 70.0,
      "job_alignment_score": 82.0,
      "presentation_score": 79.0
    },
    "strengths": [
      "Strong technical skills in required technologies",
      "Relevant work experience"
    ],
    "weaknesses": ["Missing some preferred qualifications"],
    "recommendations": [
      "Add more project details",
      "Highlight leadership experience"
    ],
    "cv_edits": [
      {
        "id": "675a1b2c3d4e5f6789012347-manual-1736858400-0",
        "field_path": "summary",
        "action": "modify",
        "current_value": "Software developer...",
        "suggested_value": "Experienced backend developer...",
        "reason": "Better alignment with job requirements",
        "priority": "high",
        "impact_score": 8.5,
        "status": ""
      }
    ],
    "jobs_analyzed": 1,
    "evaluated_at": "2026-01-14T20:30:00+07:00",
    "type": "manual",
    "job_id": "675a1b2c3d4e5f6789012345",
    "job_title": "Senior Backend Developer - ABC Company"
  }
}
```

### Response Fields

#### ResumeEvaluation Object

| Field             | Type          | Description                                          |
| ----------------- | ------------- | ---------------------------------------------------- |
| `cv_name`         | string        | Tên trên CV                                          |
| `overall_score`   | double        | Điểm tổng (0-100)                                    |
| `grade`           | string        | Xếp hạng (A+, A, B+, B, C+, C, D, F)                 |
| `score_breakdown` | object        | Chi tiết điểm từng mục                               |
| `strengths`       | array[string] | Điểm mạnh của CV                                     |
| `weaknesses`      | array[string] | Điểm yếu của CV                                      |
| `recommendations` | array[string] | Đề xuất cải thiện                                    |
| `cv_edits`        | array         | Các gợi ý chỉnh sửa CV cụ thể                        |
| `jobs_analyzed`   | int32         | Số JD đã phân tích (với manual = 1)                  |
| `evaluated_at`    | string        | Thời gian evaluate (ISO 8601)                        |
| `type`            | string        | Loại evaluation: `"auto"` hoặc `"manual"`            |
| `job_id`          | string        | ID của job được dùng để evaluate (chỉ có với manual) |
| `job_title`       | string        | Tiêu đề job + tên công ty (chỉ có với manual)        |

#### Evaluation Type

| Type     | Description                                                                           |
| -------- | ------------------------------------------------------------------------------------- |
| `auto`   | Evaluation tự động dựa trên lịch sử tương tác của user (top 2 jobs đã xem trong tuần) |
| `manual` | Evaluation do user chủ động chọn JD thông qua endpoint này                            |

---

## Lưu ý cho Frontend

1. **Flow sử dụng**:

   - Gọi `GET /api/v1/evaluation/jobs` để lấy danh sách jobs
   - Hiển thị 2 tabs/sections: "Đã xem gần đây" và "Đã lưu"
   - User chọn 1 job và 1 resume
   - Gọi `POST /api/v1/evaluation/evaluate-with-jd` với `resume_id` và `job_id`
   - Hiển thị kết quả evaluation

2. **Phân biệt evaluation type**:

   - Trong danh sách evaluations của resume, dùng field `type` để phân biệt
   - `type: "manual"` sẽ có thêm `job_id` và `job_title` để hiển thị job được chọn

3. **Loading state**:

   - API `evaluate-with-jd` là đồng bộ, có thể mất 5-15 giây
   - Nên hiển thị loading indicator phù hợp

4. **Error handling**:
   - `404 RESUME_NOT_FOUND`: Resume không tồn tại
   - `404 JOB_NOT_FOUND`: Job không tồn tại
   - `403 UNAUTHORIZED`: User không có quyền truy cập resume này
   - `500 EVALUATE_FAILED`: Lỗi từ service evaluate
