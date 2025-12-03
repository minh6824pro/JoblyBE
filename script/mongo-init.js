// MongoDB initialization script
// This script runs when MongoDB container starts for the first time
// Creates a dedicated user for the 'jobly' database

db = db.getSiblingDB('jobly');

// Create user for jobly database
db.createUser({
  user: 'jobly_user',
  pwd: 'jobly_password_2024',
  roles: [
    {
      role: 'readWrite',
      db: 'jobly'
    },
    {
      role: 'dbAdmin',
      db: 'jobly'
    }
  ]
});

print('✅ Created jobly_user with readWrite and dbAdmin roles on jobly database');

