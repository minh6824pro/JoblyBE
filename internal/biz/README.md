# Biz Layer

## Overview

The biz layer contains all business logic and use cases for the JoblyBE application.

## JD Normalizer (Ontology-based Text Normalization)

### Purpose

The `jd_normalizer.go` module provides ontology-based normalization for job descriptions before embedding into vector database. This ensures consistent vector representations regardless of how JDs are written.

### Supported Fields

1. **Skills/Technologies** - Normalizes tech stack names (e.g., "ReactJS" → "React", "Golang" → "Go")
2. **Job Titles** - Normalizes job titles and extracts category/seniority
3. **Experience Levels** - Normalizes levels with years range and responsibility context
4. **Requirements** - Normalizes and categorizes requirement text

### Tech/IT Domain Coverage

The ontology covers:

- **Programming Languages**: JavaScript, TypeScript, Python, Java, Go, C#, C++, Ruby, PHP, Rust, Kotlin, Swift, Scala, etc.
- **Frontend**: React, Vue.js, Angular, Next.js, Nuxt.js, Svelte, Tailwind CSS, Bootstrap, etc.
- **Backend**: Node.js, Express.js, NestJS, Spring Boot, Django, Flask, FastAPI, Laravel, etc.
- **Databases**: PostgreSQL, MySQL, MongoDB, Redis, Elasticsearch, DynamoDB, etc.
- **Cloud**: AWS, Azure, GCP, DigitalOcean, Vercel, Netlify
- **DevOps**: Docker, Kubernetes, Terraform, Jenkins, GitHub Actions, Prometheus, etc.
- **AI/ML**: TensorFlow, PyTorch, Scikit-learn, OpenAI, LangChain, etc.
- **Mobile**: React Native, Flutter, iOS, Android
- **Testing**: Jest, Cypress, Selenium, JUnit, PyTest
- **API**: REST, GraphQL, gRPC, WebSocket

### Usage

```go
// Get the normalizer instance
normalizer := biz.GetDefaultNormalizer()

// Normalize a job posting
normalizedData := normalizer.NormalizeJobPosting(job)

// Get texts ready for embedding
embeddingTexts := normalizer.PrepareTextForEmbedding(normalizedData)

// Use specific normalization functions
skills, categories, related := normalizer.NormalizeSkills([]string{"ReactJS", "nodejs"})
title, category, seniority := normalizer.NormalizeTitle("Senior Backend Engineer")
level, yearsRange, responsibility := normalizer.NormalizeLevel("SENIOR")
```

### Integration with Job Embedding

The `job_embedding.go` automatically uses the normalizer in `prepareEmbeddingTexts()`:

```go
// This is called automatically when embedding jobs
texts := uc.prepareEmbeddingTexts(job)
// texts["title"] = normalized title with category context
// texts["skills"] = normalized skills with related technologies
// texts["requirements"] = categorized requirements
// texts["level"] = level with years range and responsibility
```

### Adding New Skills/Titles

To add new skills or titles to the ontology, update the `initializeOntology()` method in respective ontology struct:

```go
// In SkillOntology.initializeOntology()
o.addSkill("NewSkill", "Category", []string{"alias1", "alias2"}, []string{"related1", "related2"})

// In JobTitleOntology.initializeOntology()
o.addTitle("New Title", "Category", []string{"alias1", "alias2"})
```

### Testing

Run tests to verify normalization:

```bash
go test -v ./internal/biz/... -run "Test.*Normalization|TestSkillDetection"
```
