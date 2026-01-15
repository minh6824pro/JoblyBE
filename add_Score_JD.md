/api/v1/evaluation/score-with-jd


req body
{
"job_id":"695921b9d30b32b19cce5f4b",
"resume_id": "696677d93e1126f58579a245"
}


response
{
"overallScore": 47.5,
"scoreBreakdown": {
"skillsScore": 70,
"experienceScore": 0,
"educationScore": 60,
"completenessScore": 50,
"jobAlignmentScore": 60,
"presentationScore": 75
},
"missingSkill": [
"MongoDB",
"PostgreSQL",
"Kafka",
"Microservices"
],
"jobId": "695921b9d30b32b19cce5f4b"
}


frontend gọi xong api backend sẽ lưu vào local storage cho lần sau hiển thị