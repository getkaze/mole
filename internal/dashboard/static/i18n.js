(function () {
    var translations = {
        'en': {
            /* navbar / sidebar */
            'nav.dashboard':         'Dashboard',
            'nav.developers':        'Developers',
            'nav.team':              'Team',
            'nav.modules':           'Modules',
            'nav.costs':             'Costs',
            'nav.about':             'About',
            'nav.logout':            'logout',
            'label.navigation':      'navigation',
            'label.admin':           'admin',

            /* page titles */
            'page.developers':       'Developers',
            'page.team':             'Team',
            'page.modules':          'Modules',
            'page.costs':            'Costs',
            'page.about':            'About',

            /* page subtitles */
            'page.me.subtitle':      'your review activity and growth',
            'page.dev.subtitle':     'review activity and growth',
            'page.devs.subtitle':    'individual team performance',
            'page.team.subtitle':    'quality distribution and training insights',
            'page.modules.subtitle': 'health and tech debt per module',
            'page.module.subtitle':  'health score and debt tracking',
            'page.costs.subtitle':   'Claude API usage and estimated costs',
            'page.about.subtitle':   'application settings',

            /* section labels */
            'section.overview':      'overview',
            'section.period':        'period',
            'section.files':         'Files',

            /* stat labels */
            'stat.score-trend':      'Score Trend',
            'stat.streaks':          'Streaks',
            'stat.badges':           'Badges',
            'stat.last-4-weeks':     'last 4 weeks',
            'stat.clean-prs':        'consecutive clean PRs',
            'stat.unlocked':         'achievements unlocked',
            'stat.issues':           'issues',
            'stat.debt-items':       'debt items',
            'stat.health-score':     'Health Score',
            'stat.total-issues':     'Total Issues',
            'stat.issues-found':     'issues found',
            'stat.debt-items-meta':  'tech debt items',

            /* card titles */
            'card.score-trend':      'Score Trend',
            'card.streaks-badges':   'Streaks & Badges',
            'card.issues-category':  'Issues by Category',
            'card.distribution-30d': 'distribution over the last 30 days',
            'card.weekly-90d':       'weekly trend over the last 90 days',
            'card.acceptance-rate':  'Acceptance Rate',
            'card.acceptance-subtitle': 'confirmed comments vs false positives (30 days)',
            'card.distribution-dev': 'Distribution by Developer',
            'card.last-30d':         'last 30 days',
            'card.training':         'Training Suggestions',
            'card.training-subtitle':'top issue categories across the team',

            /* table headers */
            'th.developer':          'Developer',
            'th.reviews':            'Reviews',
            'th.avg-score':          'Avg Score',
            'th.top-category':       'Top Category',
            'th.top-issue':          'Top Issue',

            /* costs */
            'costs.total':           'Total Cost',
            'costs.estimated':       'estimated in period',
            'costs.reviews':         'Reviews',
            'costs.reviews-executed': 'reviews executed',
            'costs.input-tokens':    'Input Tokens',
            'costs.input-meta':      'input tokens',
            'costs.output-tokens':   'Output Tokens',
            'costs.output-meta':     'output tokens',
            'costs.unique-prs':      'Unique PRs',
            'costs.unique-prs-meta': 'PRs reviewed',
            'costs.avg-reviews-pr':  'Avg Reviews/PR',
            'costs.avg-reviews-meta':'reviews per PR',
            'costs.avg-cost-pr':     'Avg Cost/PR',
            'costs.avg-cost-meta':   'cost per PR',
            'costs.by-model':        'by model',
            'costs.th-model':        'Model',
            'costs.th-input-cost':   'Input Cost',
            'costs.th-output-cost':  'Output Cost',
            'costs.th-total':        'Total',

            /* badges */
            'badge.attention':       'attention',
            'badge.great':           'great',
            'badge.healthy':         'healthy',
            'badge.critical':        'critical',

            /* labels */
            'label.acceptance-rate': 'acceptance rate',
            'label.confirmed':       'confirmed',
            'label.false-positive':  'false positive',
            'label.pending':         'pending',
            'label.reaction-hint':   'Devs react with 👍 or 👎 on inline comments',
            'label.streak':          'streak',
            'label.streak-desc':     'consecutive PRs without critical issues',

            /* back links */
            'link.back-modules':     '← Modules',

            /* empty states */
            'empty.no-developers':   'No developer data available.',
            'empty.no-modules':      'No module data yet. Metrics are aggregated after reviews.',
            'empty.no-issues':       'No issues found in this period.',
            'empty.no-trends':       'Insufficient data to display trends.',
            'empty.no-badges':       'Keep doing reviews to unlock badges!',
            'empty.no-team':         'No team data available yet.',
            'empty.no-validation':   'No validation data yet. Devs can react to comments with 👍 or 👎.',
            'empty.no-patterns':     'No recurring patterns detected yet.',
            'empty.no-costs':        'No usage data yet. Costs are tracked after reviews.',

            /* about page */
            'about.title':           'About Mole',
            'about.description':     'Mole is a free, self-hosted code review tool that digs deep into code and elevates those who write it.',
            'about.version':         'VERSION',
            'about.support':         'Support the Project',
            'about.support-desc':    'If Mole is useful to you, consider supporting its development.',

            /* login */
            'login.tagline':         'digs deep into code,<br>elevates those who write it.',
            'login.github-btn':      'Sign in with GitHub',
            'login.error-forbidden': 'Access restricted. You are not a member of the authorized organization.',
            'login.error-generic':   'Authentication error. Please try again.',

            /* misc */
            'loading':               'loading...',
        },
        'pt': {
            /* navbar / sidebar */
            'nav.dashboard':         'Dashboard',
            'nav.developers':        'Desenvolvedores',
            'nav.team':              'Time',
            'nav.modules':           'Módulos',
            'nav.costs':             'Custos',
            'nav.about':             'Sobre',
            'nav.logout':            'sair',
            'label.navigation':      'navegação',
            'label.admin':           'admin',

            /* page titles */
            'page.developers':       'Desenvolvedores',
            'page.team':             'Time',
            'page.modules':          'Módulos',
            'page.costs':            'Custos',
            'page.about':            'Sobre',

            /* page subtitles */
            'page.me.subtitle':      'sua atividade de review e crescimento',
            'page.dev.subtitle':     'atividade de review e crescimento',
            'page.devs.subtitle':    'desempenho individual da equipe',
            'page.team.subtitle':    'distribuição de qualidade e insights de treinamento',
            'page.modules.subtitle': 'saúde e tech debt por módulo',
            'page.module.subtitle':  'health score e debt tracking',
            'page.costs.subtitle':   'uso da API Claude e custos estimados',
            'page.about.subtitle':   'configurações da aplicação',

            /* section labels */
            'section.overview':      'visão geral',
            'section.period':        'período',
            'section.files':         'Arquivos',

            /* stat labels */
            'stat.score-trend':      'Tendência de Score',
            'stat.streaks':          'Sequências',
            'stat.badges':           'Conquistas',
            'stat.last-4-weeks':     'últimas 4 semanas',
            'stat.clean-prs':        'PRs limpos consecutivos',
            'stat.unlocked':         'conquistas desbloqueadas',
            'stat.issues':           'issues',
            'stat.debt-items':       'debt items',
            'stat.health-score':     'Health Score',
            'stat.total-issues':     'Total Issues',
            'stat.issues-found':     'issues encontrados',
            'stat.debt-items-meta':  'itens de tech debt',

            /* card titles */
            'card.score-trend':      'Tendência de Score',
            'card.streaks-badges':   'Sequências & Conquistas',
            'card.issues-category':  'Issues por Categoria',
            'card.distribution-30d': 'distribuição dos últimos 30 dias',
            'card.weekly-90d':       'evolução semanal dos últimos 90 dias',
            'card.acceptance-rate':  'Taxa de Aceite',
            'card.acceptance-subtitle': 'comentários confirmados vs falsos positivos (30 dias)',
            'card.distribution-dev': 'Distribuição por Developer',
            'card.last-30d':         'últimos 30 dias',
            'card.training':         'Sugestões de Treinamento',
            'card.training-subtitle':'top categorias de issues do time',

            /* table headers */
            'th.developer':          'Desenvolvedor',
            'th.reviews':            'Reviews',
            'th.avg-score':          'Score Médio',
            'th.top-category':       'Top Categoria',
            'th.top-issue':          'Top Issue',

            /* costs */
            'costs.total':           'Custo Total',
            'costs.estimated':       'estimado no período',
            'costs.reviews':         'Reviews',
            'costs.reviews-executed': 'revisões executadas',
            'costs.input-tokens':    'Input Tokens',
            'costs.input-meta':      'tokens de entrada',
            'costs.output-tokens':   'Output Tokens',
            'costs.output-meta':     'tokens de saída',
            'costs.unique-prs':      'PRs Únicos',
            'costs.unique-prs-meta': 'PRs revisados',
            'costs.avg-reviews-pr':  'Média Reviews/PR',
            'costs.avg-reviews-meta':'revisões por PR',
            'costs.avg-cost-pr':     'Custo Médio/PR',
            'costs.avg-cost-meta':   'custo por PR',
            'costs.by-model':        'por modelo',
            'costs.th-model':        'Modelo',
            'costs.th-input-cost':   'Custo Input',
            'costs.th-output-cost':  'Custo Output',
            'costs.th-total':        'Total',

            /* badges */
            'badge.attention':       'atenção',
            'badge.great':           'ótimo',
            'badge.healthy':         'saudável',
            'badge.critical':        'crítico',

            /* labels */
            'label.acceptance-rate': 'taxa de aceite',
            'label.confirmed':       'confirmados',
            'label.false-positive':  'falso positivo',
            'label.pending':         'pendentes',
            'label.reaction-hint':   'Devs reagem com 👍 ou 👎 nos comentários inline',
            'label.streak':          'streak',
            'label.streak-desc':     'PRs consecutivos sem issues críticos',

            /* back links */
            'link.back-modules':     '← Módulos',

            /* empty states */
            'empty.no-developers':   'Nenhum developer com dados disponíveis.',
            'empty.no-modules':      'Nenhum dado de módulo ainda. Métricas são agregadas após reviews.',
            'empty.no-issues':       'Nenhuma issue encontrada neste período.',
            'empty.no-trends':       'Dados insuficientes para exibir tendências.',
            'empty.no-badges':       'Continue fazendo reviews para desbloquear badges!',
            'empty.no-team':         'Nenhum dado de equipe disponível ainda.',
            'empty.no-validation':   'Nenhum dado de validação ainda. Devs podem reagir aos comentários com 👍 ou 👎.',
            'empty.no-patterns':     'Nenhum padrão recorrente detectado ainda.',
            'empty.no-costs':        'Nenhum dado de uso ainda. Custos são registrados após reviews.',

            /* about page */
            'about.title':           'Sobre o Mole',
            'about.description':     'Mole é uma ferramenta de code review gratuita e self-hosted que analisa código a fundo e eleva quem o escreve.',
            'about.version':         'VERSÃO',
            'about.support':         'Apoie o Projeto',
            'about.support-desc':    'Se o Mole é útil para você, considere apoiar seu desenvolvimento.',

            /* login */
            'login.tagline':         'analisa código a fundo,<br>eleva quem o escreve.',
            'login.github-btn':      'Entrar com GitHub',
            'login.error-forbidden': 'Acesso restrito. Você não é membro da organização autorizada.',
            'login.error-generic':   'Erro ao autenticar. Tente novamente.',

            /* misc */
            'loading':               'carregando...',
        }
    };

    var pageTitles = {
        'en': {
            'dashboard': 'dashboard',
            'developers': 'developers',
            'team': 'team',
            'modules': 'modules',
            'costs': 'costs',
            'about': 'about'
        },
        'pt': {
            'dashboard': 'dashboard',
            'developers': 'desenvolvedores',
            'team': 'time',
            'modules': 'módulos',
            'costs': 'custos',
            'about': 'sobre'
        }
    };

    function applyTranslations(lang) {
        var t = translations[lang] || translations['pt'];
        document.querySelectorAll('[data-i18n]').forEach(function (el) {
            var key = el.getAttribute('data-i18n');
            if (t[key] === undefined) return;
            if (t[key].indexOf('<') !== -1) {
                el.innerHTML = t[key];
            } else {
                el.textContent = t[key];
            }
        });
        document.querySelectorAll('.lang-btn').forEach(function (btn) {
            btn.classList.toggle('active', btn.getAttribute('data-lang') === lang);
        });
        document.documentElement.setAttribute('lang', lang === 'en' ? 'en' : 'pt-BR');

        // Update browser tab title
        var pageKey = document.body.getAttribute('data-page-title');
        if (pageKey) {
            var titles = pageTitles[lang] || pageTitles['pt'];
            var translated = titles[pageKey] || pageKey;
            document.title = 'Mole — ' + translated;
        }
    }

    window.setLang = function (lang) {
        localStorage.setItem('mole-lang', lang);
        applyTranslations(lang);
    };

    document.addEventListener('DOMContentLoaded', function () {
        var saved = localStorage.getItem('mole-lang') || 'pt';
        applyTranslations(saved);
    });

    // Re-apply translations after HTMX swaps new content
    document.body.addEventListener('htmx:afterSwap', function () {
        var saved = localStorage.getItem('mole-lang') || 'pt';
        applyTranslations(saved);
    });
})();
