# Dream Journal 🌙

A beautiful, secure, and personal dream journal web application built with Go and PostgreSQL. Capture your dreams, track your moods, and share select dreams with the world while maintaining complete privacy control.

![Dream Journal](https://img.shields.io/badge/Status-Active-brightgreen) ![Go](https://img.shields.io/badge/Go-1.21+-blue) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-blue) ![Security](https://img.shields.io/badge/Security-Hardened-red) ![License](https://img.shields.io/badge/License-MIT-yellow)

## ✨ Features

### **Core Features**
- **📝 Dream Recording**: Rich text editor for detailed dream descriptions
- **😊 Mood Tracking**: Optional mood selection with visual indicators (Calm, Happy, Inspired, Anxious, Haunted)
- **🔒 Privacy Control**: Choose between private and public dreams with easy toggling
- **🌐 Public Sharing**: Share inspiring dreams with others via public links
- **📱 Responsive Design**: Works beautifully on desktop and mobile devices
- **🎨 Modern UI**: Clean, light-themed interface with smooth animations
- **⚡ Fast & Lightweight**: Built with Go and PostgreSQL for optimal performance

### **Security Features**
- **🛡️ XSS Protection**: HTML sanitization using bluemonday library
- **🚦 Rate Limiting**: IP-based rate limiting to prevent abuse
- **🔐 Input Validation**: Comprehensive server-side validation
- **📋 Security Headers**: HTTP security headers for browser protection
- **🔒 SQL Injection Prevention**: Prepared statements for all database queries

## 🚀 Tech Stack

- **Backend**: Go 1.25.5+
- **Database**: PostgreSQL 15+
- **Web Framework**: Chi Router v5
- **Security**: Bluemonday (XSS prevention)
- **Frontend**: HTML5, CSS3, Vanilla JavaScript
- **Styling**: Custom CSS with modern design patterns
- **Icons**: Lucide icons
- **Fonts**: Inter & Merriweather from Google Fonts

### **Dependencies**
```go
github.com/go-chi/chi/v5 v5.2.5        // HTTP router
github.com/lib/pq v1.12.3              // PostgreSQL driver
github.com/microcosm-cc/bluemonday v1.0.25 // XSS sanitization
```

## 📦 Installation

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+ (or a hosted instance like Render PostgreSQL)
- Git

### Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/dream-journal.git
   cd dream-journal
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up PostgreSQL**
   - Create a PostgreSQL database (local or hosted)
   - Set the `DATABASE_URL` environment variable:
     ```bash
     export DATABASE_URL="postgres://username:password@host:5432/dream_journal?sslmode=require"
     ```

4. **Run the application**
   ```bash
   go run main.go
   ```

5. **Open your browser**
   ```
   http://localhost:8080
   ```

The application will automatically create the required table on first run.

## 🚢 Deploy to Render

### Step 1: Create a Render PostgreSQL Database

1. Go to [Render Dashboard](https://dashboard.render.com) → **New** → **PostgreSQL**
2. Fill in:
   - **Name**: `dream-journal-db`
   - **Database**: `dream_journal`
   - **User**: `dream_user` (or leave default)
   - **Region**: Choose the closest to you
3. Click **Create Database**
4. Wait for it to be ready (takes ~1 minute)
5. Copy the **Internal Database URL** (looks like: `postgres://user:password@host:5432/dream_journal`)

### Step 2: Deploy the Web Service

1. Push your code to a **GitHub** repository
2. In Render Dashboard → **New** → **Web Service**
3. Connect your GitHub repo
4. Fill in:
   - **Name**: `dream-journal`
   - **Region**: Same as your PostgreSQL database
   - **Branch**: `main`
   - **Runtime**: `Go`
   - **Build Command**: `go build -o lucid .`
   - **Start Command**: `./lucid`
5. Under **Environment Variables**, add:
   - **Key**: `DATABASE_URL`
   - **Value**: *(paste the Internal Database URL from Step 1)*
   - **Key**: `PORT`
   - **Value**: `10000` (Render sets this automatically, but good to have)
6. Select the **Free** or **Starter** plan
7. Click **Create Web Service**

### Step 3: Verify

1. Wait for the build & deploy to finish (~2-3 minutes)
2. Click the URL Render gives you (e.g., `https://dream-journal.onrender.com`)
3. Your Dream Journal is live! 🎉

> ⚠️ **Note**: On the free plan, the backend goes to sleep after 15 minutes of inactivity. The first request after idle will be slow (~30 seconds wake-up time).

### Step 4: Updating Your Deployment

Push new changes to GitHub — Render auto-deploys from the `main` branch. Your PostgreSQL data is **persistent** and won't be lost on redeploy.

## 🎯 Usage

### Recording Dreams

1. Click **"New Dream"** or **"+ New Dream"**
2. Fill in your dream details:
   - **Title**: Give your dream a memorable name
   - **Description**: Write the full dream story
   - **Mood**: Select from predefined moods (optional)
   - **Visibility**: Choose public or private
3. Click **"Save Dream"**

### Managing Dreams

- **View Dreams**: Browse all your recorded dreams
- **Public Dreams**: View dreams shared by the community
- **Edit Dreams**: Modify existing dream details
- **Toggle Privacy**: Switch dreams between public and private
- **Delete Dreams**: Remove dreams you no longer want

### Privacy Levels

- **Private Dreams**: Only visible to you
- **Public Dreams**: Visible to anyone with the link
- **Quick Toggle**: Change visibility anytime from dream view

## 📡 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Home page |
| GET | `/dreams` | List all dreams |
| GET | `/dreams/public` | List public dreams |
| GET | `/dreams/new` | New dream form |
| POST | `/dreams/new` | Create new dream |
| GET | `/dreams/{id}` | View single dream |
| GET | `/dreams/{id}/edit` | Edit dream form |
| POST | `/dreams/{id}` | Update dream |
| POST | `/dreams/{id}/delete` | Delete dream |

## 🛡️ Security

This application implements several security measures to protect user data:

### **Input Validation & Sanitization**
- All user inputs are validated for length and content
- HTML content is sanitized using bluemonday to prevent XSS attacks
- Form data is properly parsed and validated

### **Rate Limiting**
- IP-based rate limiting (30 requests per minute per IP)
- Prevents abuse and DoS attacks
- Thread-safe implementation with mutex protection

### **Security Headers**
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`

### **Error Handling**
- Sensitive error information is not exposed to users
- Proper logging for debugging while maintaining security

### **Database Security**
- Prepared statements prevent SQL injection
- Proper parameter binding
- PostgreSQL with SSL/TLS encryption

## 🗄️ Database Schema

```sql
CREATE TABLE dreams (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    mood TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🛠️ Development

### Project Structure

```
dream-journal/
├── main.go                 # Application entry point
├── db/
│   └── postgres.go        # PostgreSQL connection & setup
├── internal/
│   └── handler/
│       ├── dream.go       # HTTP handlers
│       └── global.go      # Global utilities
├── templates/
│   ├── index.html         # Home page
│   ├── dream.html         # Dreams list
│   ├── new.html           # Create dream form
│   ├── edit.html          # Edit dream form
│   ├── view.html          # Single dream view
│   ├── public.html        # Public dreams page
│   └── static/
│       └── style.css      # Application styles
├── go.mod                 # Go modules
├── go.sum                 # Dependency checksums
├── .gitignore            # Git ignore rules
└── README.md             # This file
```

### Building

```bash
# Build for current platform
go build -o dream-journal .

# Run tests
go test ./...

# Format code
go fmt ./...

# Run locally
DATABASE_URL="postgres://user:pass@localhost:5432/dream_journal?sslmode=disable" go run main.go
```

## 🎨 Customization

### Themes
The application uses CSS custom properties (variables) for easy theming. Modify `templates/static/style.css` to change colors:

```css
:root {
    --bg: #f8fafc;           /* Background */
    --panel: #ffffff;        /* Card backgrounds */
    --text: #1e293b;         /* Primary text */
    --brand: #3b82f6;        /* Accent color */
    /* ... more variables */
}
```

### Moods
Add new mood options by updating the HTML templates and CSS color mappings in `templates/new.html` and `templates/edit.html`.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Guidelines

- Follow Go best practices and conventions
- Write clear, descriptive commit messages
- Update documentation for new features
- Test your changes thoroughly

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Chi Router](https://github.com/go-chi/chi) for routing
- PostgreSQL driver by [lib/pq](https://github.com/lib/pq)
- Icons from [Lucide](https://lucide.dev/)
- Fonts from [Google Fonts](https://fonts.google.com/)
- Hosted on [Render](https://render.com/)

## 📞 Support

If you find this project helpful, please consider:
- ⭐ Starring the repository
- 🐛 Reporting bugs or requesting features
- 💝 Contributing code or documentation

---

**Happy Dreaming! 🌙✨**