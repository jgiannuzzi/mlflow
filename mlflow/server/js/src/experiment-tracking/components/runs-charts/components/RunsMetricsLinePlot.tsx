import { useDesignSystemTheme } from '@databricks/design-system';
import { compact, isEmpty, isEqual, maxBy, minBy } from 'lodash';
import { Config, Dash, Data as PlotlyData, Datum, Layout, LayoutAxis, TypedArray } from 'plotly.js';
import { Figure } from 'react-plotly.js';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useIntl } from 'react-intl';
import { MetricEntity } from '../../../types';
import { LazyPlot } from '../../LazyPlot';
import { useMutableChartHoverCallback } from '../hooks/useMutableHoverCallback';
import { highlightLineTraces, useRunsChartTraceHighlight } from '../hooks/useRunsChartTraceHighlight';
import {
  commonRunsChartStyles,
  RunsChartsRunData,
  runsChartDefaultMargin,
  runsChartHoverlabel,
  RunsPlotsCommonProps,
  createThemedPlotlyLayout,
  normalizeChartValue,
  useDynamicPlotSize,
  getLineChartLegendData,
  lineDashStyles,
  containsDuplicateXValues,
  createFadedTraceColor,
  getChartAxisLabelDescriptor,
  RunsChartsLineChartXAxisType,
} from './RunsCharts.common';
import { EMA } from '../../MetricsPlotView';
import RunsMetricsLegendWrapper from './RunsMetricsLegendWrapper';
import { shouldEnableDeepLearningUIPhase3, shouldEnableRelativeTimeDateAxis } from 'common/utils/FeatureUtils';
import { useRunsMultipleTracesTooltipData } from '../hooks/useRunsChartsMultipleTracesTooltip';
import { createChartImageDownloadHandler } from '../hooks/useChartImageDownloadHandler';
import {
  EPOCH_RELATIVE_TIME,
  HOUR_IN_MILLISECONDS,
  LINE_CHART_RELATIVE_TIME_THRESHOLD,
} from 'experiment-tracking/constants';

export type LineChartTraceData = PlotlyData & {
  x?: number[] | undefined;
  y?: number[];
  uuid?: string;
  metricKey?: string;
};

// Display markers only if there are less than 60 points in the single data trace
const MARKER_DISPLAY_THRESHOLD = 60;

const getDataTraceForRun = ({
  runEntry,
  metricKey,
  xAxisKey,
  selectedXAxisMetricKey,
  useDefaultHoverBox,
  lineSmoothness,
  lineShape,
  lineDash,
  displayPoints,
}: {
  runEntry: Omit<RunsChartsRunData, 'metrics' | 'params' | 'tags' | 'images'>;
  metricKey: RunsMetricsLinePlotProps['metricKey'];
  xAxisKey: RunsMetricsLinePlotProps['xAxisKey'];
  selectedXAxisMetricKey: RunsMetricsLinePlotProps['selectedXAxisMetricKey'];
  useDefaultHoverBox: RunsMetricsLinePlotProps['useDefaultHoverBox'];
  lineSmoothness: RunsMetricsLinePlotProps['lineSmoothness'];
  lineShape: RunsMetricsLinePlotProps['lineShape'];
  lineDash?: Dash;
  displayPoints?: boolean;
}): LineChartTraceData => {
  if (!runEntry.metricsHistory) {
    return {};
  }

  const sortedMetricsHistory = runEntry.metricsHistory[metricKey]?.sort((a, b) =>
    xAxisKey === RunsChartsLineChartXAxisType.STEP ? a.step - b.step : a.timestamp - b.timestamp,
  );

  let xValues;
  let yValues;
  if (xAxisKey === RunsChartsLineChartXAxisType.METRIC) {
    const xValuesWithSteps = prepareXAxisDataForMetricType(runEntry.metricsHistory[selectedXAxisMetricKey]);
    const stepOrder = xValuesWithSteps.map(({ step }) => step);
    const yValueHistory = orderBySteps(sortedMetricsHistory ?? [], stepOrder);

    xValues = xValuesWithSteps.map(({ value }) => value);
    yValues = orderBySteps(yValueHistory, stepOrder).map(({ value }) => normalizeChartValue(value));
  } else {
    xValues = prepareMetricHistoryByAxisType(sortedMetricsHistory, xAxisKey);
    yValues = sortedMetricsHistory?.map(({ value }) => normalizeChartValue(value));
  }

  // If there are any duplicate X values, use linear line shape
  // to avoid bending splines in the wrong direction
  const optimizedLineShape = containsDuplicateXValues(xValues) ? 'linear' : lineShape;

  // Use chart card's configuration or if its unset, use the automatic behavior by checking the number of points
  const shouldDisplayMarkers = displayPoints ?? xValues.length < MARKER_DISPLAY_THRESHOLD;

  const containsSingleValue = yValues?.length === 1;

  return {
    // Let's add UUID to each run so it can be distinguished later (e.g. on hover)
    uuid: runEntry.uuid,
    name: runEntry.runInfo?.runName || '',
    x: xValues,
    // The actual value is on Y axis
    y: EMA(yValues ?? [], lineSmoothness),
    // Save the metric history
    metricHistory: sortedMetricsHistory,
    metricKey,
    hovertext: runEntry.runInfo?.runName || '',
    text: 'x',
    textposition: 'outside',
    textfont: {
      size: 11,
    },
    mode: containsSingleValue || shouldDisplayMarkers ? 'lines+markers' : 'lines',
    hovertemplate: useDefaultHoverBox ? createTooltipTemplate(runEntry.runInfo?.runName || '') : undefined,
    hoverinfo: useDefaultHoverBox ? undefined : 'none',
    hoverlabel: useDefaultHoverBox ? runsChartHoverlabel : undefined,
    type: 'scatter',
    line: { dash: lineDash, shape: optimizedLineShape },
    marker: {
      color: runEntry.color,
    },
  } as LineChartTraceData;
};

const getBandTraceForRun = ({
  runEntry,
  metricKey,
  lineShape,
  xAxisKey,
  selectedXAxisMetricKey,
}: {
  runEntry: Omit<RunsChartsRunData, 'metrics' | 'params' | 'tags' | 'images'>;
  metricKey: RunsMetricsLinePlotProps['metricKey'];
  lineShape: RunsMetricsLinePlotProps['lineShape'];
  xAxisKey: RunsChartsLineChartXAxisType;
  selectedXAxisMetricKey: RunsMetricsLinePlotProps['selectedXAxisMetricKey'];
}): LineChartTraceData => {
  if (!runEntry.aggregatedMetricsHistory) {
    return {};
  }

  // Get upper and lower boundaries to draw a band
  const { max, min } = runEntry.aggregatedMetricsHistory[metricKey];

  let xMins, xMaxes, yMins, yMaxes;
  if (xAxisKey === RunsChartsLineChartXAxisType.METRIC) {
    if (!runEntry.metricsHistory) {
      return {};
    }
    const xValuesWithSteps = prepareXAxisDataForMetricType(runEntry.metricsHistory[selectedXAxisMetricKey]);
    const stepOrder = xValuesWithSteps.map((e) => e.step);
    const xValues = xValuesWithSteps.map((e) => e.value);

    yMins = orderBySteps(min, stepOrder)
      .map(({ value }) => normalizeChartValue(value))
      .reverse();
    yMaxes = orderBySteps(max, stepOrder).map(({ value }) => normalizeChartValue(value));
    xMins = xValues.slice().reverse();
    xMaxes = xValues;
  } else {
    // Reverse one of the arrays so that the band is drawn correctly
    const minReversed = min.slice().reverse();
    xMins = prepareMetricHistoryByAxisType(minReversed, xAxisKey);
    xMaxes = prepareMetricHistoryByAxisType(max, xAxisKey);
    yMins = minReversed.map(({ value }) => normalizeChartValue(value));
    yMaxes = max.map(({ value }) => normalizeChartValue(value));
  }

  // Place a null value in the middle to create a gap, otherwise Plotly will
  // connect the lines and the fill will be drawn incorrectly
  const xValues = [...xMins, null, ...xMaxes];
  const bandValues = [...yMins, null, ...yMaxes];

  return {
    name: runEntry.runInfo?.runName || '',
    x: xValues,
    y: bandValues,
    fillcolor: createFadedTraceColor(runEntry.color, 0.2),
    hovertemplate: undefined,
    hoverlabel: undefined,
    hoverinfo: 'skip',
    line: { color: 'transparent', shape: lineShape },
    fill: 'tozeroy',
    type: 'scatter',
  } as LineChartTraceData;
};

/**
 * This function takes a list of metric entities and returns a copy ordered by
 * the step order provided. This is used in metric-type X axes, where the Y values
 * need to be ordered by the X values.
 *
 * For example:
 * dataPoints = [{step: 0, value: 1}, {step: 1, value: 2}, {step: 2, value: 3}]
 * stepOrder = [2, 0, 1]
 * return = [{step: 2, value: 3}, {step: 0, value: 1}, {step: 1, value: 2}]
 */
const orderBySteps = (dataPoints: MetricEntity[], stepOrder: number[]) => {
  const stepIndexes = stepOrder.reduce((acc, step, idx) => {
    acc[step] = idx;
    return acc;
  }, {} as Record<number, number>);

  return dataPoints.slice().sort((a, b) => stepIndexes[a.step] - stepIndexes[b.step]);
};

export interface RunsMetricsSingleTraceTooltipData {
  xValue: string | number;
  yValue: number;
  index: number;
  label: string;
  traceUuid?: string;
  metricEntity?: MetricEntity;
}

export interface RunsCompareMultipleTracesTooltipData {
  tooltipLegendItems: {
    uuid: string;
    color?: string;
    dashStyle?: Dash;
    displayName: string;
    value?: string | number;
  }[];
  xValue: string | number;
  xAxisKey: RunsChartsLineChartXAxisType;
  xAxisKeyLabel: string;
  hoveredDataPoint?: RunsMetricsSingleTraceTooltipData;
}

export interface RunsMetricsLinePlotProps extends RunsPlotsCommonProps {
  /**
   * Determines which metric are we comparing by
   * NOTE: used only as a fallback in V2 charts
   */
  metricKey: string;

  /**
   * Determines which metric keys to display in V2 charts
   * NOTE: this prop may not be present in V1 chart configs
   */
  selectedMetricKeys?: string[];

  /**
   * Smoothing factor for EMA
   */
  lineSmoothness?: number;

  /**
   * X axis mode
   */
  xAxisScaleType?: 'linear' | 'log';

  /**
   * Y axis mode
   */
  scaleType?: 'linear' | 'log';

  /**
   * Use spline or polyline. Spline is default mode.
   */
  lineShape?: 'linear' | 'spline';

  /**
   * Choose X axis mode - numeric step or absolute time
   */
  xAxisKey?: RunsChartsLineChartXAxisType;

  /**
   * Name of the metric to use for the X axis. Used when xAxisKey is set to 'metric'
   */
  selectedXAxisMetricKey: string;

  /**
   * Array of runs data with corresponding values
   */
  runsData: Omit<RunsChartsRunData, 'metrics' | 'params' | 'tags' | 'images'>[];

  /**
   * Currently visible range on x-axis.
   */
  xRange?: [number | string, number | string];

  /**
   * Currently visible range on y-axis
   */
  yRange?: [number | string, number | string];

  /**
   * If set to true, only x-axis can be zoomed by dragging
   */
  lockXAxisZoom?: boolean;

  /**
   * Display points on the line chart. Undefined means "auto" mode, i.e. display points only when
   * there are fewer than 60 datapoints on the chart.
   */
  displayPoints?: boolean;
}

const PLOT_CONFIG: Partial<Config> = {
  displaylogo: false,
  doubleClick: 'autosize',
  scrollZoom: false,
  modeBarButtonsToRemove: ['toImage'],
};

export const createTooltipTemplate = (runName: string) =>
  `<b>${runName}</b>:<br>` +
  '<b>%{xaxis.title.text}:</b> %{x}<br>' +
  '<b>%{yaxis.title.text}:</b> %{y:.2f}<br>' +
  '<extra></extra>';

/**
 * Prepares dataset's X axis according to selected visualization type: step, time-wall and time-relative
 *
 * NOTE: metric-type X axes are handled by `prepareXAxisDataForMetricType()`, since we need to retain
 *       step information in order to format the Y axis data correctly.
 */
const prepareMetricHistoryByAxisType = (
  metricHistory?: MetricEntity[],
  axisType?: RunsMetricsLinePlotProps['xAxisKey'],
) => {
  if (!metricHistory) {
    return [];
  }
  if (axisType === RunsChartsLineChartXAxisType.TIME_RELATIVE) {
    const { timestamp: minTimestamp } = minBy(metricHistory, 'timestamp') || {};
    if (minTimestamp) {
      if (shouldEnableRelativeTimeDateAxis()) {
        return metricHistory.map(({ timestamp }) => timestamp - minTimestamp + EPOCH_RELATIVE_TIME);
      }
      return metricHistory.map(({ timestamp }) => (timestamp - minTimestamp) / 1000); // Milliseconds -> seconds
    }
    return metricHistory.map(({ step }) => step);
  } else if (shouldEnableRelativeTimeDateAxis() && axisType === RunsChartsLineChartXAxisType.TIME_RELATIVE_HOURS) {
    const { timestamp: minTimestamp } = minBy(metricHistory, 'timestamp') || {};
    if (minTimestamp) {
      return metricHistory.map(({ timestamp }) => (timestamp - minTimestamp) / HOUR_IN_MILLISECONDS);
    }
  } else if (axisType === RunsChartsLineChartXAxisType.TIME) {
    return metricHistory.map(({ timestamp }) => timestamp);
  }

  return metricHistory.map(({ step }) => step);
};

/**
 * Prepares dataset's X axis when axisType is 'metric'. This is separate from
 * `prepareMetricHistoryByAxisType` because we need to keep track of the `step`
 * in addition to the `value`, so that the Y axis data can be associated to the
 * correct X datapoint.
 */
const prepareXAxisDataForMetricType = (
  metricHistory?: MetricEntity[],
): Array<{
  value: number | undefined;
  step: number;
}> => {
  if (!metricHistory) {
    return [];
  }

  return metricHistory
    .map((datapoint) => ({
      value: normalizeChartValue(datapoint.value),
      step: datapoint.step,
    }))
    .sort((a, b) => {
      // sort by value in ascending order
      return Number(a.value) - Number(b.value);
    });
};

/**
 * Implementation of plotly.js chart displaying
 * line plot comparing metrics' history for a given
 * set of experiments runs
 */
export const RunsMetricsLinePlot = React.memo(
  ({
    runsData,
    metricKey,
    selectedMetricKeys,
    scaleType = 'linear',
    xAxisScaleType = 'linear',
    xAxisKey = RunsChartsLineChartXAxisType.STEP,
    selectedXAxisMetricKey = '',
    lineSmoothness = 70,
    className,
    margin = runsChartDefaultMargin,
    lineShape = 'linear',
    onUpdate,
    onHover,
    onUnhover,
    width,
    height,
    useDefaultHoverBox = true,
    selectedRunUuid,
    xRange,
    yRange,
    lockXAxisZoom,
    fullScreen,
    displayPoints,
    onSetDownloadHandler,
  }: RunsMetricsLinePlotProps) => {
    const { theme } = useDesignSystemTheme();
    const usingMultipleRunsHoverTooltip = shouldEnableDeepLearningUIPhase3();

    const dynamicXAxisKey = useMemo(() => {
      let dynamicXAxisKey = xAxisKey;
      if (shouldEnableRelativeTimeDateAxis() && xAxisKey === RunsChartsLineChartXAxisType.TIME_RELATIVE) {
        const metricKeys = selectedMetricKeys || [metricKey];
        let maxDiff = 0;
        runsData.forEach((runData) => {
          const metricHistory = runData.metricsHistory;
          if (metricHistory) {
            metricKeys.forEach((metricKey) => {
              if (metricHistory[metricKey]) {
                const { timestamp: minTimestamp } = minBy(metricHistory[metricKey], 'timestamp') || {};
                const { timestamp: maxTimestamp } = maxBy(metricHistory[metricKey], 'timestamp') || {};
                if (maxTimestamp && minTimestamp) {
                  const diff = maxTimestamp - minTimestamp;
                  maxDiff = Math.max(maxDiff, diff);
                }
              }
            });
          }
        });

        if (maxDiff >= LINE_CHART_RELATIVE_TIME_THRESHOLD) {
          dynamicXAxisKey = RunsChartsLineChartXAxisType.TIME_RELATIVE_HOURS;
        }
      }
      return dynamicXAxisKey;
    }, [runsData, selectedMetricKeys, metricKey, xAxisKey]);

    const plotData = useMemo(() => {
      // Generate a data trace for each metric in each run
      const metricKeys = selectedMetricKeys ?? [metricKey];
      return runsData
        .map((runEntry) =>
          metricKeys
            // Discard creating traces for metrics that don't have any history for a given run
            .filter((metricKey) => !isEmpty(runEntry.metricsHistory?.[metricKey]))
            .map((metricKey, idx) => {
              return getDataTraceForRun({
                runEntry,
                metricKey,
                xAxisKey: dynamicXAxisKey,
                selectedXAxisMetricKey,
                useDefaultHoverBox,
                lineSmoothness,
                lineShape,
                lineDash: lineDashStyles[idx % lineDashStyles.length],
                displayPoints,
              });
            }),
        )
        .flat();
    }, [
      runsData,
      lineShape,
      dynamicXAxisKey,
      lineSmoothness,
      metricKey,
      useDefaultHoverBox,
      selectedMetricKeys,
      selectedXAxisMetricKey,
      displayPoints,
    ]);

    const bandsData = useMemo(() => {
      const metricKeys = selectedMetricKeys ?? [metricKey];
      return runsData
        .filter(({ groupParentInfo }) => groupParentInfo)
        .flatMap((runEntry) =>
          metricKeys.map((metricKey) =>
            getBandTraceForRun({
              runEntry,
              metricKey,
              lineShape,
              xAxisKey: dynamicXAxisKey,
              selectedXAxisMetricKey,
            }),
          ),
        );
    }, [lineShape, metricKey, runsData, selectedMetricKeys, dynamicXAxisKey, selectedXAxisMetricKey]);

    const plotDataWithBands = useMemo(() => [...bandsData, ...plotData], [plotData, bandsData]);

    const { layoutHeight, layoutWidth, setContainerDiv, containerDiv, isDynamicSizeSupported } = useDynamicPlotSize();

    const { formatMessage } = useIntl();

    const { setHoveredPointIndex } = useRunsChartTraceHighlight(
      containerDiv,
      selectedRunUuid,
      plotDataWithBands,
      highlightLineTraces,
      bandsData.length,
    );

    const xAxisKeyLabel = useMemo(() => {
      if (dynamicXAxisKey === RunsChartsLineChartXAxisType.METRIC) {
        return selectedXAxisMetricKey;
      }

      return formatMessage(getChartAxisLabelDescriptor(dynamicXAxisKey));
    }, [formatMessage, dynamicXAxisKey, selectedXAxisMetricKey]);

    const yAxisParams: Partial<LayoutAxis> = useMemo(
      () => ({
        tickfont: { size: 11, color: theme.colors.textSecondary },
        type: scaleType === 'log' ? 'log' : 'linear',
        fixedrange: lockXAxisZoom,
      }),
      [scaleType, lockXAxisZoom, theme],
    );

    const [layout, setLayout] = useState<Partial<Layout>>({
      width: width || layoutWidth,
      height: height || layoutHeight,
      margin,
      xaxis: {
        title: xAxisKeyLabel,
        tickfont: { size: 11, color: theme.colors.textSecondary },
      },
      yaxis: yAxisParams,
      showlegend: false,
    });

    useEffect(() => {
      setLayout((current) => {
        const updatedLayout = {
          ...current,
          width: width || layoutWidth,
          height: height || layoutHeight,
          margin,
          yaxis: yAxisParams,
          showlegend: false,
        };
        if (isEqual(updatedLayout, current)) {
          return current;
        }
        return updatedLayout;
      });
    }, [layoutWidth, layoutHeight, margin, yAxisParams, width, height, xAxisKeyLabel]);

    const containsMultipleMetricKeys = useMemo(() => (selectedMetricKeys?.length || 0) > 1, [selectedMetricKeys]);

    const hoverCallback = useCallback(
      ({ points, event }) => {
        const hoveredPoint = points[0];
        const hoveredPointData = hoveredPoint?.data;
        setHoveredPointIndex(hoveredPoint?.curveNumber ?? -1);

        if (!hoveredPointData) {
          return;
        }
        const runUuid = hoveredPointData.uuid;

        // Extract metric entity
        const metricEntity = hoveredPointData.metricHistory?.[hoveredPoint.pointIndex];

        const data: RunsMetricsSingleTraceTooltipData = {
          // Value of the "x" axis (time, step)
          xValue: hoveredPoint.x,
          // Value of the "y" axis
          yValue: hoveredPoint.y,
          // Metric entity value
          metricEntity,
          // The index of the X datum
          index: hoveredPoint.pointIndex,
          // Current label ("Step", "Time" etc.)
          label: xAxisKeyLabel,
        };
        if (runUuid) {
          onHover?.(runUuid, event, data);
        }
      },
      [onHover, setHoveredPointIndex, xAxisKeyLabel],
    );

    const unhoverCallback = useCallback(() => {
      onUnhover?.();
      setHoveredPointIndex(-1);
    }, [onUnhover, setHoveredPointIndex]);

    const themedPlotlyLayout = useMemo(() => createThemedPlotlyLayout(theme), [theme]);

    const getXAxisType = (xAxisKey: RunsChartsLineChartXAxisType, xAxisScaleType: 'linear' | 'log') => {
      if (
        xAxisKey === RunsChartsLineChartXAxisType.TIME ||
        (shouldEnableRelativeTimeDateAxis() && dynamicXAxisKey === RunsChartsLineChartXAxisType.TIME_RELATIVE)
      ) {
        return 'date';
      }
      if (xAxisKey === RunsChartsLineChartXAxisType.STEP && xAxisScaleType === 'log') {
        return 'log';
      }
      return 'linear';
    };

    const getXAxisRange = (
      xAxisKey: RunsChartsLineChartXAxisType,
      xRange: [number | string, number | string],
      xAxisScaleType: 'linear' | 'log',
    ) => {
      if (
        xAxisKey === RunsChartsLineChartXAxisType.STEP &&
        typeof xRange[0] === 'number' &&
        typeof xRange[1] === 'number'
      ) {
        if (xAxisScaleType === 'log') {
          if (xRange[0] < 0 && xRange[1] < 0) {
            // If both are negative, autoscale
            return undefined;
          } else if (xRange[0] < 0) {
            // If only the lower bound is negative, set it to 0
            return [0, xRange[1]];
          }
        }
      }
      return [...xRange];
    };

    // When switching axis title, Plotly.js mutates its layout object
    // internally which leads to desync problems and automatic axis range
    // ends up with an invalid value. In order to fix it, we are mutating
    // axis object and injecting metric key as title in
    // the render phase.
    // It could be fixed by wrapping plotly.js directly instead of using
    // react-plotly.js - but the effort does not correspond to the plan of
    // moving away from plotly soon.
    const immediateLayout = layout;
    if (immediateLayout.xaxis) {
      immediateLayout.xaxis.title = xAxisKeyLabel;
      immediateLayout.xaxis.type = getXAxisType(xAxisKey, xAxisScaleType);
      if (xRange) {
        immediateLayout.xaxis.range = getXAxisRange(xAxisKey, xRange, xAxisScaleType);
      }
      immediateLayout.xaxis.tickformat =
        shouldEnableRelativeTimeDateAxis() && dynamicXAxisKey === RunsChartsLineChartXAxisType.TIME_RELATIVE
          ? '%H:%M:%S'
          : undefined;
    }
    immediateLayout.template = { layout: themedPlotlyLayout };

    if (yRange && immediateLayout.yaxis) {
      immediateLayout.yaxis.range = yRange;
      immediateLayout.yaxis.automargin = true;
      immediateLayout.yaxis.tickformat = 'f';
    }

    const legendLabelData = useMemo(
      () => getLineChartLegendData(runsData, selectedMetricKeys, metricKey),
      [runsData, selectedMetricKeys, metricKey],
    );

    const {
      scanlineElement,
      initHandler,
      updateHandler: updateHandlerMultipleRuns,
      onPointHover: hoverCallbackMultipleRuns,
      onPointUnhover: unhoverCallbackMultipleRuns,
    } = useRunsMultipleTracesTooltipData({
      legendLabelData,
      plotData,
      runsData,
      containsMultipleMetricKeys,
      onHover,
      onUnhover: unhoverCallback,
      xAxisKeyLabel,
      xAxisKey: dynamicXAxisKey,
      xAxisScaleType: xAxisKey === RunsChartsLineChartXAxisType.STEP ? xAxisScaleType : 'linear',
      setHoveredPointIndex,
      disabled: !usingMultipleRunsHoverTooltip,
    });

    /**
     * Unfortunately plotly.js memorizes first onHover callback given on initial render,
     * so in order to achieve updated behavior we need to wrap its most recent implementation
     * in the immutable callback.
     */
    const mutableHoverCallback = useMutableChartHoverCallback(
      usingMultipleRunsHoverTooltip ? hoverCallbackMultipleRuns : hoverCallback,
    );

    // Prepare data for image download handler
    useEffect(() => {
      // Check if we are using multiple metric keys. If so, we also need to append
      // the metric key to  the trace name in the exported image.
      const usingMultipleMetricKeys = (selectedMetricKeys?.length || 0) > 1;
      const dataToExport = usingMultipleMetricKeys
        ? plotDataWithBands.map((dataTrace) =>
            dataTrace.metricKey
              ? {
                  ...dataTrace,
                  name: `${dataTrace.name} (${dataTrace.metricKey})`,
                }
              : dataTrace,
          )
        : plotDataWithBands;

      const layoutToExport: Partial<Layout> = {
        ...layout,
        showlegend: true,
        legend: {
          orientation: 'h',
        },
      };
      onSetDownloadHandler?.(createChartImageDownloadHandler(dataToExport, layoutToExport));
    }, [layout, onSetDownloadHandler, plotDataWithBands, selectedMetricKeys?.length]);

    const chart = (
      <div
        css={[commonRunsChartStyles.chartWrapper(theme), styles.highlightStyles]}
        className={className}
        ref={setContainerDiv}
      >
        <LazyPlot
          data={plotDataWithBands}
          useResizeHandler={!isDynamicSizeSupported}
          css={commonRunsChartStyles.chart(theme)}
          onUpdate={(figure: Readonly<Figure>, graphDiv: Readonly<HTMLElement>) => {
            if (usingMultipleRunsHoverTooltip) {
              updateHandlerMultipleRuns(figure, graphDiv);
            }
            onUpdate?.(figure, graphDiv);
          }}
          layout={immediateLayout}
          config={PLOT_CONFIG}
          onHover={mutableHoverCallback}
          onUnhover={usingMultipleRunsHoverTooltip ? unhoverCallbackMultipleRuns : unhoverCallback}
          onInitialized={initHandler}
        />
        {scanlineElement}
      </div>
    );

    return (
      <RunsMetricsLegendWrapper labelData={legendLabelData} fullScreen={fullScreen}>
        {chart}
      </RunsMetricsLegendWrapper>
    );
  },
);

const styles = {
  highlightStyles: {
    '.scatterlayer g.trace': {
      transition: 'var(--trace-transition)',
    },
    '.scatterlayer.is-highlight g.trace:not(.is-band)': {
      opacity: 'var(--trace-opacity-dimmed-low) !important',
    },
    '.scatterlayer g.trace.is-hover-highlight:not(.is-band)': {
      opacity: 'var(--trace-opacity-highlighted) !important',
    },
    '.scatterlayer g.trace.is-selection-highlight:not(.is-band)': {
      opacity: 'var(--trace-opacity-highlighted) !important',
    },
    '.scatterlayer g.trace.is-selection-highlight path.point': {
      stroke: 'var(--trace-stroke-color)',
      strokeWidth: 'var(--trace-stroke-width) !important',
    },
    '.scatterlayer.is-highlight g.trace.is-band:not(.is-band-highlighted)': {
      opacity: 'var(--trace-opacity-dimmed) !important',
    },
  },
};
