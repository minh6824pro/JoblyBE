// MongoDB seed script for companies and job postings
// Run with: mongosh <database_name> seed_data.js
use("jobly");

// Clear existing data (optional - remove if you want to keep existing data)
db.companies.deleteMany({});
db.job_postings.deleteMany({});

// Insert Companies
const companies = [
  {
    _id: ObjectId(),
    name: "One Mount",
    description:
      "Leading Fintech company in Vietnam, providing diversified payment solutions and developing digital financial platforms.",
    website: "https://onemount.com/",
    logo_url: "https://nodeflair.com/api/v2/companies/9114.png",
    industry: "Fintech",
    company_size: "1001+",
    location: "Vietnam",
    founded_year: "2018",
    created_at: new Date(),
    updated_at: new Date(),
  },
  {
    _id: ObjectId(),
    name: "MB Bank",
    description: "One of the leading commercial banks in Vietnam",
    website: "https://www.mbbank.com.vn/",
    logo_url: "",
    industry: "Banking",
    company_size: "1001+",
    location: "Vietnam",
    founded_year: "1994",
    created_at: new Date(),
    updated_at: new Date(),
  },
  {
    _id: ObjectId(),
    name: "Techcombank",
    description: "Vietnam Technological and Commercial Joint Stock Bank",
    website: "https://www.techcombank.com.vn/",
    logo_url: "https://nodeflair.com/api/v2/companies/8467.png",
    industry: "Banking",
    company_size: "1001+",
    location: "Vietnam",
    founded_year: "1993",
    created_at: new Date(),
    updated_at: new Date(),
  },
  {
    _id: ObjectId(),
    name: "Capgemini",
    description:
      "Global leader in consulting, technology services and digital transformation",
    website: "https://www.capgemini.com/",
    logo_url: "https://nodeflair.com/api/v2/companies/375.png",
    industry: "Technology Consulting",
    company_size: "1001+",
    location: "Vietnam",
    founded_year: "1967",
    created_at: new Date(),
    updated_at: new Date(),
  },
  {
    _id: ObjectId(),
    name: "Bosch",
    description: "Leading global supplier of technology and services",
    website: "https://www.bosch.com/",
    logo_url: "https://nodeflair.com/api/v2/companies/141.png",
    industry: "Manufacturing & Technology",
    company_size: "1001+",
    location: "Vietnam",
    founded_year: "1886",
    created_at: new Date(),
    updated_at: new Date(),
  },
  {
    _id: ObjectId(),
    name: "SSI Securities",
    description: "Leading securities company in Vietnam",
    website: "https://www.ssi.com.vn/",
    logo_url: "https://nodeflair.com/api/v2/companies/13046.png",
    industry: "Securities & Finance",
    company_size: "501-1000",
    location: "Vietnam",
    founded_year: "1999",
    created_at: new Date(),
    updated_at: new Date(),
  },
];

// Insert companies and store their IDs
const insertedCompanies = db.companies.insertMany(companies);
const companyIds = Object.values(insertedCompanies.insertedIds);

print("✅ Inserted " + companyIds.length + " companies");

// Insert Job Postings
const jobPostings = [
  {
    company_id: companyIds[0], // One Mount
    title: "Senior Backend Engineer (Java)",
    level: "SENIOR",
    job_type: "FULL_TIME",
    salary_min: 30278805,
    salary_max: 61143224,
    salary_currency: "VND",
    location: "Vietnam",
    posted_at: new Date(new Date().setMonth(new Date().getMonth() - 5)),
    experience_requirement:
      "5+ years of experience with Java and Spring Framework",
    description:
      "We are looking for experienced Senior Java Developer to join our product team. This is a fantastic opportunity to work at One Mount Group. As a member of One Mount - VinID Pay, we aim to become a leading Fintech company in Vietnam, providing diversified payment solutions and developing digital financial platforms.",
    responsibilities:
      "• Writing clean & high-quality code\n• Maintain & improve existing systems, design and develop new features\n• Deliver end-to-end solutions including testing and deployment\n• Participate in code reviews and ensure coding quality standards",
    requirements:
      "• 5+ years of experience with Java and Spring Framework (Spring Boot)\n• Strong knowledge of Data Structures and Algorithms\n• Experience with SQL/NoSQL (MySQL, PostgreSQL, MongoDB)\n• Experience with REST APIs, Microservices\n• Familiarity with Redis, Kafka\n• Experience with Docker/Kubernetes and Cloud Infrastructure is a plus\n• Experience with Agile/Scrum methodologies",
    benefits:
      "• 13th month salary and annual bonuses\n• Lunch allowance 730,000 VND/month\n• Special occasion bonuses (2,500,000 VND/year)\n• Up to 20 annual leave days\n• Premium health insurance, yearly health check\n• Laptop and tools provided\n• Career growth opportunities and learning resources (Udemy, Coursera)\n• Open, collaborative, and young working environment with unwind zones and team building events",
    job_tech: [
      "MongoDB",
      "Redis",
      "MySQL",
      "PostgreSQL",
      "SQL",
      "Spring Boot",
      "Kafka",
      "Kubernetes",
      "NoSQL",
      "Java",
      "Spring",
      "Docker",
      "REST API",
    ],
    created_at: new Date(),
  },
  {
    company_id: companyIds[1], // MB Bank
    title:
      "Chuyên Viên Kiểm Thử Nghiệp Vụ - Manual Tester - Khối Công Nghệ Thông Tin (HOLT.06)",
    level: "MID",
    job_type: "FULL_TIME",
    salary_min: 0,
    salary_max: 0,
    salary_currency: "VND",
    location: "Vietnam",
    posted_at: new Date(new Date().setMonth(new Date().getMonth() - 6)),
    experience_requirement: "Trên 2 năm kinh nghiệm kiểm thử",
    description:
      "• Tiếp nhận các yêu cầu kiểm thử các sản phẩm, dịch vụ Công nghệ Thông tin\n• Thực hiện tìm hiểu, phân tích và làm rõ yêu cầu. Xây dựng chiến lược kiểm thử cho yêu cầu sản phẩm, dịch vụ Công nghệ thông tin\n• Xây dựng kế hoạch kiểm thử, viết kịch bản, tạo báo cáo kiểm thử, báo cáo lỗi kiểm thử. Xây dựng tài liệu cho các yêu cầu sau khi đã thực hiện\n• Thực hiện kiểm thử, đánh giá chất lượng sản phẩm\n• Phân tích / đánh giá nguyên nhân lỗi sau golive",
    responsibilities:
      "• Tiếp nhận các yêu cầu kiểm thử\n• Thực hiện kiểm thử sản phẩm, đánh giá chất lượng\n• Xây dựng báo cáo, tài liệu liên quan",
    requirements:
      "• Tốt nghiệp Đại học\n• Chuyên ngành: CNTT, Toán tin, Điện tử Viễn thông, Kế Toán, Tài chính Ngân hàng...\n• Kiến thức về ngân hàng và nghiệp vụ ngân hàng\n• Hiểu biết sâu về kiểm thử phần mềm\n• Trên 2 năm kinh nghiệm kiểm thử\n• Kinh nghiệm Agile/Scrum\n• Kinh nghiệm kiểm thử Android/iOS\n• Kỹ năng quản lý nhóm và tư duy logic\n• Tiếng Anh tốt, giao tiếp với đối tác nước ngoài",
    benefits:
      "• Nhanh nhẹn, trung thực, tin cậy\n• Môi trường làm việc chuyên nghiệp, cơ hội phát triển và học tập",
    job_tech: ["Android", "iOS"],
    created_at: new Date(),
  },
  {
    company_id: companyIds[2], // Techcombank
    title: "Expert/Senior Java Developer",
    level: "SENIOR",
    job_type: "FULL_TIME",
    salary_min: 25099300,
    salary_max: 65216750,
    salary_currency: "VND",
    location: "Vietnam",
    posted_at: new Date(new Date().setMonth(new Date().getMonth() - 6)),
    experience_requirement: "8-10 years relevant experience in Java",
    description:
      "I. Key Accountabilities\n\nA. Software Development\n• Responsible for ensuring that the bank's digital platforms work well by managing back end site databases, performance issues, security and that the server, application and database communicate with each other.\n• Responsible for collaborating with business tribes to understand the needs and technical requirements before building a web application.\n• Responsible for the server-side web application logic and integration with front-end codes\n• Collaborate with Front End Developers to design more functional and cohesive codes to enhance user experience.\n• Responsible for driving the application lifecycle with a key focus in coding and debugging of web applications based on feedback from testers and users.\n• Compile and analyze data, process and codes to troubleshoot problems and identify areas for improvement.\n\nB. Software Documentation\n• Work closely with tribe and squad members to translate business requirements into technical design documents.\n• Review and implement technical requirement documents by coding flowcharts, layouts, diagrams, charts, code comments and guides for the program.",
    responsibilities:
      "• Responsible for ensuring backend databases, performance, security\n• Collaborate with Front End Developers\n• Drive application lifecycle, coding, debugging\n• Compile and analyze data to troubleshoot problems",
    requirements:
      "• 8-10 years relevant experience in Java\n• SQL, microservices, AWS experience\n• Database: Oracle, Microsoft SQL Server, PostgreSQL\n• Linux/Redhat environment\n• CI/CD, logging, monitoring platforms\n• Agile/Scrum experience\n• Bachelor's degree in CS, Software Engineering, IT\n• Good English\n• Insurance experience is preferable",
    benefits:
      "• Passionate about technology\n• Ownership mindset\n• Self-starter, innovative\n• Collaborative and transparent culture",
    job_tech: [
      "PostgreSQL",
      "SQL",
      "Linux",
      "Java",
      "Microsoft T SQL",
      "CIS",
      "Spring",
      "Microsoft SQL Server",
      "Oracle",
      "AWS",
    ],
    created_at: new Date(),
  },
  {
    company_id: companyIds[3], // Capgemini
    title: "Internship Tester (French)",
    level: "ENTRY",
    job_type: "INTERNSHIP",
    salary_min: 0,
    salary_max: 0,
    salary_currency: "VND",
    location: "Vietnam",
    posted_at: new Date(new Date().setMonth(new Date().getMonth() - 6)),
    experience_requirement: "No experience required",
    description:
      "About the Role/position:\nIn this role, you will be familiar with tester position. You will be trained on testing techniques and then work alongside senior testers within the Testing community. One should be a phenomenal teammate with a forward-thinking mindset, ability, and confidence to challenge the status quo to define future visions.",
    responsibilities:
      "• Work alongside senior testers\n• Learn testing techniques\n• Collaborate in the Testing community",
    requirements:
      "Primary Skills: Programming Language: Any Object Oriented Programming language like Java/Python/C# (applied for IT Degree)\nQualifications: Degree in CS, IT, or other majors with good French communication\n• Be able to work fulltime",
    benefits:
      "• Receiving technical training courses\n• Good opportunity to be a good tester in the future\n• Opportunity to work on international projects\n• Professional and dynamic working environment\n• Gain valuable experience with various projects, new technologies, and talents",
    job_tech: ["Python", "C#", "Java", "OOP"],
    created_at: new Date(),
  },
  {
    company_id: companyIds[1], // MB Bank
    title:
      "Chuyên Viên, Chuyên Viên Cao Cấp Khoa Học Dữ Liệu - Data Scientist - Khối Dữ Liệu (HO25.92)",
    level: "MID",
    job_type: "FULL_TIME",
    salary_min: 0,
    salary_max: 0,
    salary_currency: "VND",
    location: "Vietnam",
    posted_at: new Date(new Date().setMonth(new Date().getMonth() - 7)),
    experience_requirement:
      "Tối thiểu 3 năm làm việc với dữ liệu lớn, xây dựng mô hình ML",
    description:
      "• Phân tích dữ liệu chuyên sâu để xây dựng mô hình phục vụ các mảng nghiệp vụ được phân công\n• Thiết kế và phát triển các đặc trưng (features) từ dữ liệu, làm giàu kho lưu trữ dữ liệu đặc trưng.\n• Giám sát, tối ưu hóa và vận hành việc xây dựng và phát triển mô hình để đáp ứng nhu cầu kinh doanh và hỗ trợ ra quyết định kinh doanh.\n• Nghiên cứu và phát triển công cụ, quy trình nhằm tăng hiệu quả hoạt động của phòng/khối.\n• Báo cáo trực tiếp lãnh đạo về tiến độ và kết quả các dự án phân tích dữ liệu được giao",
    responsibilities:
      "• Phân tích dữ liệu, xây dựng và vận hành mô hình\n• Giám sát tiến độ dự án, tối ưu hóa mô hình\n• Báo cáo kết quả và đề xuất cải tiến",
    requirements:
      "• Cử nhân Kinh tế, Tài chính, Ngân hàng, Khoa học Dữ liệu, CNTT, Thống kê, Toán tin hoặc liên quan\n• Ưu tiên chứng chỉ về Data Science/Engineering/Analysis\n• Thành thạo Python, R, SQL, Matlab\n• Kinh nghiệm công cụ dữ liệu lớn: Spark, Hadoop, S3\n• Tối thiểu 3 năm làm việc với dữ liệu lớn, xây dựng mô hình ML\n• Hiểu biết thuật toán ML, khai phá dữ liệu, phát triển thuật toán\n• Kỹ năng trực quan hóa dữ liệu: Matplotlib, Tableau, storytelling từ dữ liệu\n• Ưu tiên ứng viên có kinh nghiệm quản lý",
    benefits:
      "• Môi trường làm việc chuyên nghiệp\n• Cơ hội phát triển nghề nghiệp và kỹ năng Data Science\n• Tham gia dự án lớn, ứng dụng công nghệ hiện đại",
    job_tech: [
      "Python",
      "Hadoop",
      "SQL",
      "Spark",
      "MATLAB",
      "Matplotlib",
      "R",
      "Amazon S3",
    ],
    created_at: new Date(),
  },
  {
    company_id: companyIds[4], // Bosch
    title: "Automation Software Tester (Selenium/java/appium)",
    level: "MID",
    job_type: "FULL_TIME",
    salary_min: 0,
    salary_max: 0,
    salary_currency: "VND",
    location: "Vietnam",
    posted_at: new Date(new Date().setMonth(new Date().getMonth() - 7)),
    experience_requirement: "1-2 years experience in automation testing",
    description:
      "- Develop, maintain and execute automation test cases for major projects, maintenance, and emergency releases\n- Design and implement automation tests scripts, debug and define corrective actions\n- Identify, analyze and report test results\n- Report, track, and monitor defects in the defect tracking system\n- Investigate defect reports from production support, isolate their causes, inform development teams for fixing and retest to ensure adequate resolutions\n- Work closely with the PO and development teams to design testing strategies\n- Work on the interpretation of quality assurance issues and problems for technical and non-technical users",
    responsibilities:
      "- Develop, maintain, and execute automation test cases\n- Design and implement automation scripts, debug and correct\n- Track and report defects\n- Collaborate with PO and development teams",
    requirements:
      "- 1-2 years experience in automation testing (web, API, mobile) with Selenium, Appium (Java, C#)\n- At least one programming or scripting language (C#, Java)\n- Experience writing test cases based on requirements\n- Ability to manage multiple tasks and priorities\n- Cross-browser, cross-platform, responsive testing experience\n- Familiar with source version control tools\n- Problem-solving and analytical skills\n- Willing to learn new technologies and testing methodologies",
    benefits:
      "• Môi trường làm việc chuyên nghiệp, cơ hội phát triển kỹ năng automation testing\n• Tham gia dự án đa dạng với công nghệ hiện đại",
    job_tech: ["C#", "Appium", "Java", "API", "Selenium"],
    created_at: new Date(),
  },
  {
    company_id: companyIds[5], // SSI Securities
    title: "Chuyên Viên Phân Tích Dữ Liệu (Data Analyst)",
    level: "MID",
    job_type: "FULL_TIME",
    salary_min: 0,
    salary_max: 0,
    salary_currency: "VND",
    location: "Vietnam",
    posted_at: new Date(new Date().setMonth(new Date().getMonth() - 7)),
    experience_requirement: "Ít nhất 2 năm kinh nghiệm Data Analyst",
    description:
      "1. Phân tích & trực quan hóa dữ liệu:\n- Thu thập, xử lý, và phân tích dữ liệu từ nhiều nguồn khác nhau để hỗ trợ ra quyết định kinh doanh.\n- Xây dựng các báo cáo, dashboard trực quan hóa dữ liệu để hỗ trợ hoạt động kinh doanh/vận hành.\n- Theo dõi và đánh giá hiệu quả hoạt động dựa trên các chỉ số kinh doanh (KPIs).\n\n2. Xây dựng & quản lý cơ sở dữ liệu:\n- Thiết kế, tối ưu hóa các mô hình dữ liệu phục vụ phân tích.\n- Viết truy vấn SQL để trích xuất, tổng hợp dữ liệu từ nhiều nguồn dữ liệu khác nhau (Datawarehouse/DataLake/ hoặc các nguồn khác).\n- Hỗ trợ xây dựng và quản lý pipeline dữ liệu tự động.\n\n3. Hỗ trợ phân tích chuyên sâu:\n- Phân tích xu hướng thị trường, hành vi khách hàng, hiệu quả chiến dịch để đề xuất giải pháp tối ưu.\n- Hỗ trợ các phòng ban trong việc sử dụng dữ liệu để đưa ra quyết định kinh doanh.\n\n4. Phối hợp với các bộ phận khác:\n- Làm việc chặt chẽ với các team IT, Kinh doanh, Marketing, Rủi ro để đảm bảo dữ liệu được sử dụng hiệu quả.\n- Đề xuất các phương pháp phân tích dữ liệu mới nhằm nâng cao chất lượng dự báo.\n\n5. Thực hiện các công việc chuyên môn liên quan khác theo phân giao của cấp quản lý",
    responsibilities:
      "• Phân tích & trực quan hóa dữ liệu\n• Xây dựng & quản lý cơ sở dữ liệu\n• Hỗ trợ phân tích chuyên sâu\n• Phối hợp với các bộ phận khác\n• Thực hiện các công việc chuyên môn liên quan",
    requirements:
      "• Tốt nghiệp Đại học chuyên ngành: Khoa học dữ liệu, Toán học, Kinh tế, Tài chính, CNTT hoặc liên quan\n• Ít nhất 2 năm kinh nghiệm Data Analyst, ưu tiên tài chính/chứng khoán\n• Hiểu biết thị trường chứng khoán, Big Data, Spark, Hadoop\n• Thành thạo SQL (MS SQL, Oracle,…)\n• Kinh nghiệm Power BI, Tableau\n• Kỹ năng lập trình Python\n• Hiểu biết Data Warehouse, DataLake, ETL\n• Kỹ năng tư duy phân tích, giải quyết vấn đề",
    benefits:
      "• Môi trường làm việc chuyên nghiệp\n• Cơ hội phát triển kỹ năng Data Analyst\n• Tham gia dự án dữ liệu lớn với công nghệ hiện đại",
    job_tech: [
      "Python",
      "Hadoop",
      "SQL",
      "Spark",
      "MSSQL",
      "Oracle",
      "DataLake",
      "ETL",
    ],
    created_at: new Date(),
  },
];

// Insert job postings
const insertedJobs = db.job_postings.insertMany(jobPostings);
print(
  "✅ Inserted " +
    Object.keys(insertedJobs.insertedIds).length +
    " job postings"
);

// Create indexes for better query performance
db.companies.createIndex({ name: 1 });
db.companies.createIndex({ industry: 1 });
db.companies.createIndex({ location: 1 });

db.job_postings.createIndex({ company_id: 1 });
db.job_postings.createIndex({ title: "text", description: "text" });
db.job_postings.createIndex({ level: 1 });
db.job_postings.createIndex({ job_type: 1 });
db.job_postings.createIndex({ location: 1 });
db.job_postings.createIndex({ job_tech: 1 });
db.job_postings.createIndex({ posted_at: -1 });
db.job_postings.createIndex({ created_at: -1 });

print("✅ Created indexes");
print("\n🎉 Seed data completed successfully!");
print("📊 Total companies: " + db.companies.countDocuments());
print("📊 Total job postings: " + db.job_postings.countDocuments());
