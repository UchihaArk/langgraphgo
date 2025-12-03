document.addEventListener('DOMContentLoaded', () => {
    // Add fade-in class to elements we want to animate
    const animatedElements = document.querySelectorAll('.card, .section-title, .hero-cta');
    animatedElements.forEach(el => {
        el.classList.add('fade-in-section');
    });

    const observerOptions = {
        root: null,
        rootMargin: '0px',
        threshold: 0.1
    };

    const observer = new IntersectionObserver((entries, observer) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('is-visible');
                observer.unobserve(entry.target); // Only animate once
            }
        });
    }, observerOptions);

    document.querySelectorAll('.fade-in-section').forEach(section => {
        observer.observe(section);
    });

    // Header scroll effect
    const header = document.querySelector('header');
    window.addEventListener('scroll', () => {
        if (window.scrollY > 50) {
            header.style.background = 'rgba(255, 255, 255, 0.9)';
            header.style.boxShadow = 'var(--shadow-sm)';
        } else {
            header.style.background = 'var(--bg-glass)';
            header.style.boxShadow = 'none';
        }
    });
});
