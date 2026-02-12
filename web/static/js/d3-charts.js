/**
 * Loom D3.js Visualization Library
 *
 * Reusable, animated chart components:
 *  - Donut / ring charts (provider share, agent workload)
 *  - Animated horizontal bar charts with gradient fills
 *  - Sparklines (latency trend, token usage over time)
 *  - Gauge arcs (error rate, capacity)
 *  - Radial status rings (agent health)
 *  - Treemaps (bead distribution)
 */

/* global d3 */
/* exported LoomCharts */

const LoomCharts = (function () {
    'use strict';

    // ── Palette ──────────────────────────────────────────────────────
    const PALETTE = [
        '#2563eb', '#7c3aed', '#059669', '#d97706', '#dc2626',
        '#0891b2', '#4f46e5', '#16a34a', '#ea580c', '#be185d'
    ];

    const STATUS_COLORS = {
        working: '#16a34a',
        idle: '#2563eb',
        paused: '#d97706',
        error: '#dc2626',
        blocked: '#dc2626',
        healthy: '#16a34a',
        active: '#16a34a',
        pending: '#d97706',
        failed: '#dc2626',
        open: '#2563eb',
        in_progress: '#7c3aed',
        closed: '#64748b',
        done: '#059669'
    };

    function statusColor(status) {
        return STATUS_COLORS[(status || '').toLowerCase()] || '#94a3b8';
    }

    function pickColor(i) {
        return PALETTE[i % PALETTE.length];
    }

    // ── Helpers ──────────────────────────────────────────────────────
    function clearEl(el) {
        while (el.firstChild) {
            el.removeChild(el.firstChild);
        }
    }

    function ensureSVG(container, width, height, margin) {
        clearEl(container);
        var m = margin || { top: 10, right: 10, bottom: 10, left: 10 };
        var svg = d3.select(container)
            .append('svg')
            .attr('width', width)
            .attr('height', height)
            .append('g')
            .attr('transform', 'translate(' + m.left + ',' + m.top + ')');
        return { svg: svg, width: width - m.left - m.right, height: height - m.top - m.bottom };
    }

    function shortNum(n) {
        if (n >= 1e9) { return (n / 1e9).toFixed(1) + 'B'; }
        if (n >= 1e6) { return (n / 1e6).toFixed(1) + 'M'; }
        if (n >= 1e3) { return (n / 1e3).toFixed(1) + 'K'; }
        return String(n);
    }

    function truncLabel(s, max) {
        if (!s) { return ''; }
        return s.length > max ? s.slice(0, max - 1) + '\u2026' : s;
    }

    // ── Donut Chart ─────────────────────────────────────────────────
    //  data: [{label, value, color?}]  or  {label: value, ...}
    function donut(container, rawData, opts) {
        opts = opts || {};
        var size = opts.size || 220;
        var thickness = opts.thickness || 36;
        var data = normalizeKV(rawData);
        if (!data.length) {
            container.innerHTML = '<p class="small" style="text-align:center;padding:2rem;color:var(--text-muted)">No data</p>';
            return;
        }

        clearEl(container);

        var radius = size / 2;
        var arc = d3.arc().innerRadius(radius - thickness).outerRadius(radius);
        var pie = d3.pie().value(function (d) { return d.value; }).sort(null).padAngle(0.02);

        var svg = d3.select(container)
            .append('svg')
            .attr('width', size)
            .attr('height', size)
            .append('g')
            .attr('transform', 'translate(' + radius + ',' + radius + ')');

        var total = d3.sum(data, function (d) { return d.value; });

        var arcs = svg.selectAll('.arc')
            .data(pie(data))
            .enter()
            .append('g')
            .attr('class', 'arc');

        arcs.append('path')
            .attr('d', arc)
            .attr('fill', function (d, i) { return d.data.color || pickColor(i); })
            .attr('opacity', 0.9)
            .style('cursor', 'pointer')
            .on('mouseover', function () { d3.select(this).attr('opacity', 1).attr('transform', 'scale(1.04)'); })
            .on('mouseout', function () { d3.select(this).attr('opacity', 0.9).attr('transform', 'scale(1)'); })
            .transition()
            .duration(800)
            .attrTween('d', function (d) {
                var interp = d3.interpolate({ startAngle: 0, endAngle: 0 }, d);
                return function (t) { return arc(interp(t)); };
            });

        // Center label
        svg.append('text')
            .attr('text-anchor', 'middle')
            .attr('dy', '-0.2em')
            .style('font-size', '1.5rem')
            .style('font-weight', '700')
            .style('fill', 'var(--text-color)')
            .text(shortNum(total));

        svg.append('text')
            .attr('text-anchor', 'middle')
            .attr('dy', '1.2em')
            .style('font-size', '0.7rem')
            .style('fill', 'var(--text-muted)')
            .text(opts.centerLabel || 'total');

        // Legend below
        var legend = d3.select(container)
            .append('div')
            .style('display', 'flex')
            .style('flex-wrap', 'wrap')
            .style('gap', '0.5rem 1rem')
            .style('margin-top', '0.75rem')
            .style('justify-content', 'center');

        data.forEach(function (d, i) {
            var item = legend.append('span')
                .style('display', 'inline-flex')
                .style('align-items', 'center')
                .style('gap', '0.3rem')
                .style('font-size', '0.8rem')
                .style('color', 'var(--text-muted)');

            item.append('span')
                .style('width', '10px')
                .style('height', '10px')
                .style('border-radius', '50%')
                .style('background', d.color || pickColor(i))
                .style('flex-shrink', '0');

            item.append('span').text(truncLabel(d.label, 18) + ' (' + shortNum(d.value) + ')');
        });
    }

    // ── Animated Horizontal Bar Chart ───────────────────────────────
    function barChart(container, rawData, opts) {
        opts = opts || {};
        var data = normalizeKV(rawData);
        if (!data.length) {
            container.innerHTML = '<p class="small" style="text-align:center;padding:2rem;color:var(--text-muted)">No data</p>';
            return;
        }

        data.sort(function (a, b) { return b.value - a.value; });

        var barH = opts.barHeight || 28;
        var gap = 6;
        var labelW = opts.labelWidth || 130;
        var valueW = 70;
        var margin = { top: 4, right: valueW + 8, bottom: 4, left: labelW };
        var totalH = data.length * (barH + gap) + margin.top + margin.bottom;
        var cw = container.clientWidth || 400;

        clearEl(container);

        var ctx = ensureSVG(container, cw, totalH, margin);
        var maxVal = d3.max(data, function (d) { return d.value; }) || 1;
        var x = d3.scaleLinear().domain([0, maxVal]).range([0, ctx.width]);

        // Gradient definition
        var defs = d3.select(container).select('svg').append('defs');
        data.forEach(function (d, i) {
            var grad = defs.append('linearGradient')
                .attr('id', 'bar-grad-' + i)
                .attr('x1', '0%').attr('x2', '100%');
            var c = d.color || pickColor(i);
            grad.append('stop').attr('offset', '0%').attr('stop-color', c).attr('stop-opacity', 0.85);
            grad.append('stop').attr('offset', '100%').attr('stop-color', d3.color(c).brighter(0.5)).attr('stop-opacity', 1);
        });

        var rows = ctx.svg.selectAll('.bar-row')
            .data(data)
            .enter()
            .append('g')
            .attr('class', 'bar-row')
            .attr('transform', function (d, i) { return 'translate(0,' + i * (barH + gap) + ')'; });

        // Labels
        rows.append('text')
            .attr('x', -8)
            .attr('y', barH / 2)
            .attr('dy', '0.35em')
            .attr('text-anchor', 'end')
            .style('font-size', '0.8rem')
            .style('fill', 'var(--text-color)')
            .text(function (d) { return truncLabel(d.label, 16); });

        // Background track
        rows.append('rect')
            .attr('width', ctx.width)
            .attr('height', barH)
            .attr('rx', 4)
            .attr('fill', 'var(--light-bg)');

        // Animated fill
        rows.append('rect')
            .attr('height', barH)
            .attr('rx', 4)
            .attr('fill', function (d, i) { return 'url(#bar-grad-' + i + ')'; })
            .attr('width', 0)
            .transition()
            .duration(700)
            .delay(function (d, i) { return i * 60; })
            .attr('width', function (d) { return x(d.value); });

        // Value labels
        rows.append('text')
            .attr('x', ctx.width + 8)
            .attr('y', barH / 2)
            .attr('dy', '0.35em')
            .style('font-size', '0.8rem')
            .style('font-weight', '600')
            .style('fill', 'var(--text-muted)')
            .text(function (d) {
                var prefix = opts.prefix || '';
                return prefix + shortNum(d.value);
            });
    }

    // ── Sparkline ───────────────────────────────────────────────────
    //  data: [number] or [{value, label?}]
    function sparkline(container, rawData, opts) {
        opts = opts || {};
        var values = rawData.map(function (d) { return typeof d === 'number' ? d : d.value; });
        if (!values.length) { return; }

        var w = opts.width || container.clientWidth || 120;
        var h = opts.height || 32;
        var color = opts.color || '#2563eb';

        clearEl(container);

        var svg = d3.select(container)
            .append('svg')
            .attr('width', w)
            .attr('height', h);

        var x = d3.scaleLinear().domain([0, values.length - 1]).range([2, w - 2]);
        var y = d3.scaleLinear().domain([d3.min(values) * 0.9, d3.max(values) * 1.1]).range([h - 2, 2]);

        var area = d3.area()
            .x(function (d, i) { return x(i); })
            .y0(h)
            .y1(function (d) { return y(d); })
            .curve(d3.curveCatmullRom);

        var line = d3.line()
            .x(function (d, i) { return x(i); })
            .y(function (d) { return y(d); })
            .curve(d3.curveCatmullRom);

        // Gradient fill under line
        var defs = svg.append('defs');
        var grad = defs.append('linearGradient')
            .attr('id', 'spark-grad-' + Math.random().toString(36).slice(2, 8))
            .attr('x1', '0').attr('x2', '0').attr('y1', '0').attr('y2', '1');
        grad.append('stop').attr('offset', '0%').attr('stop-color', color).attr('stop-opacity', 0.25);
        grad.append('stop').attr('offset', '100%').attr('stop-color', color).attr('stop-opacity', 0);
        var gradId = grad.attr('id');

        svg.append('path')
            .datum(values)
            .attr('fill', 'url(#' + gradId + ')')
            .attr('d', area);

        svg.append('path')
            .datum(values)
            .attr('fill', 'none')
            .attr('stroke', color)
            .attr('stroke-width', 1.5)
            .attr('d', line);

        // End dot
        svg.append('circle')
            .attr('cx', x(values.length - 1))
            .attr('cy', y(values[values.length - 1]))
            .attr('r', 2.5)
            .attr('fill', color);
    }

    // ── Gauge Arc ───────────────────────────────────────────────────
    //  value: 0..1,  label: string
    function gauge(container, value, opts) {
        opts = opts || {};
        var size = opts.size || 120;
        var thickness = opts.thickness || 12;
        var color = opts.color || (value > 0.8 ? '#dc2626' : value > 0.5 ? '#d97706' : '#16a34a');

        clearEl(container);

        var radius = size / 2;
        var tau = 2 * Math.PI;
        var arcGen = d3.arc().innerRadius(radius - thickness).outerRadius(radius).startAngle(-tau / 4);

        var svg = d3.select(container)
            .append('svg')
            .attr('width', size)
            .attr('height', size * 0.65)
            .append('g')
            .attr('transform', 'translate(' + radius + ',' + radius + ')');

        // Background arc
        svg.append('path')
            .datum({ endAngle: tau / 4 })
            .attr('d', arcGen)
            .attr('fill', '#e2e8f0');

        // Value arc with animation
        svg.append('path')
            .datum({ endAngle: -tau / 4 })
            .attr('fill', color)
            .transition()
            .duration(800)
            .attrTween('d', function () {
                var target = -tau / 4 + (tau / 2) * Math.min(value, 1);
                var interp = d3.interpolate(-tau / 4, target);
                return function (t) {
                    return arcGen({ endAngle: interp(t) });
                };
            });

        // Center text
        svg.append('text')
            .attr('text-anchor', 'middle')
            .attr('dy', '0.1em')
            .style('font-size', '1.1rem')
            .style('font-weight', '700')
            .style('fill', 'var(--text-color)')
            .text(opts.label || Math.round(value * 100) + '%');
    }

    // ── Radial Status Ring ──────────────────────────────────────────
    //  statuses: {working: 4, idle: 2, paused: 1}
    function statusRing(container, statuses, opts) {
        opts = opts || {};
        var size = opts.size || 160;
        var thickness = opts.thickness || 24;
        var data = [];
        var total = 0;
        Object.keys(statuses).forEach(function (k) {
            var v = statuses[k];
            if (v > 0) {
                data.push({ label: k, value: v, color: statusColor(k) });
                total += v;
            }
        });
        if (!data.length) { return; }

        clearEl(container);

        var radius = size / 2;
        var arc = d3.arc().innerRadius(radius - thickness).outerRadius(radius);
        var pie = d3.pie().value(function (d) { return d.value; }).sort(null).padAngle(0.03);

        var svg = d3.select(container)
            .append('svg')
            .attr('width', size)
            .attr('height', size)
            .append('g')
            .attr('transform', 'translate(' + radius + ',' + radius + ')');

        svg.selectAll('.arc')
            .data(pie(data))
            .enter()
            .append('path')
            .attr('fill', function (d) { return d.data.color; })
            .attr('opacity', 0.9)
            .transition()
            .duration(800)
            .attrTween('d', function (d) {
                var interp = d3.interpolate({ startAngle: 0, endAngle: 0 }, d);
                return function (t) { return arc(interp(t)); };
            });

        svg.append('text')
            .attr('text-anchor', 'middle')
            .attr('dy', '-0.1em')
            .style('font-size', '1.6rem')
            .style('font-weight', '700')
            .style('fill', 'var(--text-color)')
            .text(total);

        svg.append('text')
            .attr('text-anchor', 'middle')
            .attr('dy', '1.3em')
            .style('font-size', '0.7rem')
            .style('fill', 'var(--text-muted)')
            .text(opts.centerLabel || 'agents');

        // Inline legend
        var legend = d3.select(container)
            .append('div')
            .style('display', 'flex')
            .style('gap', '0.75rem')
            .style('margin-top', '0.5rem')
            .style('justify-content', 'center')
            .style('flex-wrap', 'wrap');

        data.forEach(function (d) {
            var item = legend.append('span')
                .style('display', 'inline-flex')
                .style('align-items', 'center')
                .style('gap', '0.25rem')
                .style('font-size', '0.75rem');

            item.append('span')
                .style('width', '8px')
                .style('height', '8px')
                .style('border-radius', '50%')
                .style('background', d.color);

            item.append('span')
                .style('color', 'var(--text-muted)')
                .text(d.label + ' ' + d.value);
        });
    }

    // ── Mini Treemap ────────────────────────────────────────────────
    //  data: [{label, value, color?, status?}]
    function treemap(container, rawData, opts) {
        opts = opts || {};
        var data = normalizeKV(rawData);
        if (!data.length) {
            container.innerHTML = '<p class="small" style="text-align:center;padding:2rem;color:var(--text-muted)">No data</p>';
            return;
        }

        var w = opts.width || container.clientWidth || 400;
        var h = opts.height || 200;

        clearEl(container);

        var root = d3.hierarchy({ children: data }).sum(function (d) { return d.value; });

        d3.treemap()
            .size([w, h])
            .padding(2)
            .round(true)(root);

        var svg = d3.select(container)
            .append('svg')
            .attr('width', w)
            .attr('height', h);

        var cell = svg.selectAll('g')
            .data(root.leaves())
            .enter()
            .append('g')
            .attr('transform', function (d) { return 'translate(' + d.x0 + ',' + d.y0 + ')'; });

        cell.append('rect')
            .attr('width', function (d) { return d.x1 - d.x0; })
            .attr('height', function (d) { return d.y1 - d.y0; })
            .attr('rx', 3)
            .attr('fill', function (d, i) { return d.data.color || (d.data.status ? statusColor(d.data.status) : pickColor(i)); })
            .attr('opacity', 0.85)
            .style('cursor', 'pointer')
            .on('mouseover', function () { d3.select(this).attr('opacity', 1); })
            .on('mouseout', function () { d3.select(this).attr('opacity', 0.85); });

        cell.append('text')
            .attr('x', 4)
            .attr('y', 14)
            .style('font-size', '0.7rem')
            .style('fill', '#fff')
            .style('font-weight', '600')
            .style('pointer-events', 'none')
            .text(function (d) {
                var cellW = d.x1 - d.x0;
                if (cellW < 40) { return ''; }
                return truncLabel(d.data.label, Math.floor(cellW / 7));
            });

        cell.append('text')
            .attr('x', 4)
            .attr('y', 28)
            .style('font-size', '0.65rem')
            .style('fill', 'rgba(255,255,255,0.8)')
            .style('pointer-events', 'none')
            .text(function (d) {
                var cellW = d.x1 - d.x0;
                if (cellW < 50) { return ''; }
                return shortNum(d.value);
            });

        // Tooltip on hover via title
        cell.append('title')
            .text(function (d) { return d.data.label + ': ' + shortNum(d.value); });
    }

    // ── Stat Card with animated counter ─────────────────────────────
    function animateCounter(el, target, opts) {
        opts = opts || {};
        var duration = opts.duration || 800;
        var prefix = opts.prefix || '';
        var suffix = opts.suffix || '';
        var decimals = opts.decimals || 0;
        var start = parseFloat(el.textContent.replace(/[^0-9.-]/g, '')) || 0;

        d3.select(el)
            .transition()
            .duration(duration)
            .tween('text', function () {
                var interp = d3.interpolateNumber(start, target);
                return function (t) {
                    var v = interp(t);
                    this.textContent = prefix + (decimals ? v.toFixed(decimals) : Math.round(v).toLocaleString()) + suffix;
                };
            });
    }

    // ── Utility: normalize data formats ─────────────────────────────
    function normalizeKV(raw) {
        if (Array.isArray(raw)) {
            return raw.map(function (d) {
                if (typeof d === 'object' && d.label !== undefined) { return d; }
                return { label: String(d), value: d };
            });
        }
        if (raw && typeof raw === 'object') {
            return Object.keys(raw).map(function (k) {
                return { label: k, value: raw[k] };
            });
        }
        return [];
    }

    // ── Public API ──────────────────────────────────────────────────
    return {
        donut: donut,
        barChart: barChart,
        sparkline: sparkline,
        gauge: gauge,
        statusRing: statusRing,
        treemap: treemap,
        animateCounter: animateCounter,
        statusColor: statusColor,
        pickColor: pickColor,
        shortNum: shortNum
    };
})();
