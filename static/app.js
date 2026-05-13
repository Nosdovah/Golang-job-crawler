document.addEventListener('DOMContentLoaded', () => {
    const crawlBtn = document.getElementById('crawlBtn');
    const btnText = crawlBtn.querySelector('.btn-text');
    const spinner = crawlBtn.querySelector('.spinner');
    
    const dashboard = document.getElementById('dashboard');
    const emptyState = document.getElementById('emptyState');
    
    const totalJobsEl = document.getElementById('totalJobs');
    const statsListEl = document.getElementById('statsList');
    const matchesListEl = document.getElementById('matchesList');
    const jobsListEl = document.getElementById('jobsList');

    crawlBtn.addEventListener('click', async () => {
        // UI Loading State
        btnText.textContent = "Scanning...";
        spinner.classList.remove('hidden');
        crawlBtn.disabled = true;
        
        try {
            const res = await fetch('/api/crawl');
            const data = await res.json();
            
            // Switch views
            emptyState.classList.add('hidden');
            dashboard.classList.remove('hidden');
            
            // Populate Data
            populateDashboard(data);
            
        } catch (error) {
            console.error("Error crawling jobs:", error);
            alert("Failed to reach the crawler backend.");
        } finally {
            // Restore UI State
            btnText.textContent = "Rescan Market";
            spinner.classList.add('hidden');
            crawlBtn.disabled = false;
        }
    });

    function populateDashboard(data) {
        // Animate total jobs number
        animateValue(totalJobsEl, 0, data.total_jobs, 1000);
        
        // 1. Stats
        statsListEl.innerHTML = '';
        if (data.stats && data.stats.length > 0) {
            data.stats.forEach(stat => {
                const item = document.createElement('div');
                item.className = 'stat-item';
                item.innerHTML = `
                    <div class="stat-name">${stat.requirement}</div>
                    <div class="stat-bar-container">
                        <div class="stat-bar" style="width: 0%"></div>
                    </div>
                    <div class="stat-value">${stat.percentage}%</div>
                `;
                statsListEl.appendChild(item);
                
                // Trigger animation next frame
                requestAnimationFrame(() => {
                    setTimeout(() => {
                        item.querySelector('.stat-bar').style.width = `${stat.percentage}%`;
                    }, 100);
                });
            });
        } else {
            statsListEl.innerHTML = '<p class="subtitle">Not enough data to calculate stats.</p>';
        }

        // 2. Matches
        matchesListEl.innerHTML = '';
        if (data.matches && data.matches.length > 0) {
            data.matches.forEach(match => {
                const tags = match.shared_stack.map(s => `<span class="tag">${s}</span>`).join('');
                
                const card = document.createElement('div');
                card.className = 'match-card';
                card.innerHTML = `
                    <div class="match-header">
                        <div class="match-score">${match.score}% Match</div>
                    </div>
                    <div class="job-role"><strong>A:</strong> ${match.job1.title} <span>@ ${match.job1.company}</span></div>
                    <div class="job-role"><strong>B:</strong> ${match.job2.title} <span>@ ${match.job2.company}</span></div>
                    <div class="shared-stack">
                        ${tags}
                    </div>
                `;
                matchesListEl.appendChild(card);
            });
        } else {
            matchesListEl.innerHTML = '<p class="subtitle">No highly similar roles found in this sample.</p>';
        }

        // 3. Raw Jobs
        jobsListEl.innerHTML = '';
        if (data.jobs && data.jobs.length > 0) {
            data.jobs.slice(0, 100).forEach(job => { // Show up to 100 jobs
                const card = document.createElement('div');
                card.className = 'raw-job-card';
                card.innerHTML = `
                    <h3>${job.title}</h3>
                    <div class="meta">${job.company} • ${job.location}</div>
                    <div style="margin-bottom:10px;">
                        ${job.requirements ? job.requirements.map(r => `<span class="tag">${r}</span>`).join(' ') : ''}
                    </div>
                    <a href="${job.url}" target="_blank">View Original Posting →</a>
                `;
                jobsListEl.appendChild(card);
            });
        }
    }

    // Utility: Number animation
    function animateValue(obj, start, end, duration) {
        let startTimestamp = null;
        const step = (timestamp) => {
            if (!startTimestamp) startTimestamp = timestamp;
            const progress = Math.min((timestamp - startTimestamp) / duration, 1);
            obj.innerHTML = Math.floor(progress * (end - start) + start);
            if (progress < 1) {
                window.requestAnimationFrame(step);
            }
        };
        window.requestAnimationFrame(step);
    }
});
