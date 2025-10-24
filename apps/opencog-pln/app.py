"""
OpenCog PLN (Probabilistic Logic Networks) Microservice

This service provides probabilistic reasoning and inference capabilities.
PLN operates on knowledge stored in AtomSpace to perform logical inference.
"""

from flask import Flask, request, jsonify
from flask_cors import CORS
import logging
import os
from datetime import datetime
from typing import Dict, List, Any, Tuple
import json
import math

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)


class TruthValue:
    """Represents a truth value with strength and confidence"""
    
    def __init__(self, strength: float = 0.5, confidence: float = 0.5):
        self.strength = max(0.0, min(1.0, strength))
        self.confidence = max(0.0, min(1.0, confidence))
    
    def to_dict(self) -> Dict[str, float]:
        return {
            'strength': self.strength,
            'confidence': self.confidence
        }
    
    @staticmethod
    def from_dict(data: Dict[str, float]) -> 'TruthValue':
        return TruthValue(
            strength=data.get('strength', 0.5),
            confidence=data.get('confidence', 0.5)
        )


class PLNInference:
    """Implements basic PLN inference rules"""
    
    @staticmethod
    def deduction(premise1: TruthValue, premise2: TruthValue) -> TruthValue:
        """
        Deduction rule: If A->B and B->C, then A->C
        Simple implementation using strength multiplication
        """
        strength = premise1.strength * premise2.strength
        confidence = min(premise1.confidence, premise2.confidence)
        return TruthValue(strength, confidence)
    
    @staticmethod
    def induction(premise1: TruthValue, premise2: TruthValue) -> TruthValue:
        """
        Induction rule: Generalize from specific instances
        """
        strength = (premise1.strength + premise2.strength) / 2
        confidence = min(premise1.confidence, premise2.confidence) * 0.8
        return TruthValue(strength, confidence)
    
    @staticmethod
    def abduction(premise1: TruthValue, premise2: TruthValue) -> TruthValue:
        """
        Abduction rule: Hypothesis generation
        """
        strength = (premise1.strength * premise2.strength) ** 0.5
        confidence = min(premise1.confidence, premise2.confidence) * 0.7
        return TruthValue(strength, confidence)
    
    @staticmethod
    def conjunction(premise1: TruthValue, premise2: TruthValue) -> TruthValue:
        """
        Conjunction rule: A AND B
        """
        strength = premise1.strength * premise2.strength
        confidence = (premise1.confidence + premise2.confidence) / 2
        return TruthValue(strength, confidence)
    
    @staticmethod
    def disjunction(premise1: TruthValue, premise2: TruthValue) -> TruthValue:
        """
        Disjunction rule: A OR B
        """
        strength = premise1.strength + premise2.strength - (premise1.strength * premise2.strength)
        confidence = (premise1.confidence + premise2.confidence) / 2
        return TruthValue(strength, confidence)
    
    @staticmethod
    def negation(premise: TruthValue) -> TruthValue:
        """
        Negation rule: NOT A
        """
        return TruthValue(1.0 - premise.strength, premise.confidence)
    
    @staticmethod
    def revision(tv1: TruthValue, tv2: TruthValue) -> TruthValue:
        """
        Revision rule: Combine two truth values for the same statement
        """
        # Weighted average based on confidence
        total_confidence = tv1.confidence + tv2.confidence
        if total_confidence == 0:
            return TruthValue(0.5, 0.0)
        
        strength = (tv1.strength * tv1.confidence + tv2.strength * tv2.confidence) / total_confidence
        confidence = min(1.0, total_confidence)
        return TruthValue(strength, confidence)


# PLN state
pln_state = {
    'inference_history': [],
    'rules_applied': 0
}


@app.route('/healthcheck', methods=['GET'])
def healthcheck():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'opencog-pln',
        'timestamp': datetime.utcnow().isoformat(),
        'rules_applied': pln_state['rules_applied']
    }), 200


@app.route('/v1/version', methods=['GET'])
def version():
    """Version information endpoint"""
    return jsonify({
        'service': 'opencog-pln',
        'version': '0.1.0',
        'opencog_version': 'compatible',
        'description': 'OpenCog Probabilistic Logic Networks reasoning service'
    }), 200


@app.route('/api/v1/infer/deduction', methods=['POST'])
def infer_deduction():
    """
    Apply deduction inference rule
    
    Request body:
    {
        "premise1": {"strength": 0.9, "confidence": 0.8},
        "premise2": {"strength": 0.8, "confidence": 0.7}
    }
    """
    try:
        data = request.get_json()
        
        if not data or 'premise1' not in data or 'premise2' not in data:
            return jsonify({
                'error': 'Missing required fields: premise1 and premise2'
            }), 400
        
        tv1 = TruthValue.from_dict(data['premise1'])
        tv2 = TruthValue.from_dict(data['premise2'])
        
        result = PLNInference.deduction(tv1, tv2)
        
        inference = {
            'rule': 'deduction',
            'premises': [tv1.to_dict(), tv2.to_dict()],
            'result': result.to_dict(),
            'timestamp': datetime.utcnow().isoformat()
        }
        
        pln_state['inference_history'].append(inference)
        pln_state['rules_applied'] += 1
        
        logger.info(f"Applied deduction rule: {result.to_dict()}")
        
        return jsonify({
            'success': True,
            'inference': inference
        }), 200
        
    except Exception as e:
        logger.error(f"Error in deduction inference: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/infer/induction', methods=['POST'])
def infer_induction():
    """Apply induction inference rule"""
    try:
        data = request.get_json()
        
        if not data or 'premise1' not in data or 'premise2' not in data:
            return jsonify({
                'error': 'Missing required fields: premise1 and premise2'
            }), 400
        
        tv1 = TruthValue.from_dict(data['premise1'])
        tv2 = TruthValue.from_dict(data['premise2'])
        
        result = PLNInference.induction(tv1, tv2)
        
        inference = {
            'rule': 'induction',
            'premises': [tv1.to_dict(), tv2.to_dict()],
            'result': result.to_dict(),
            'timestamp': datetime.utcnow().isoformat()
        }
        
        pln_state['inference_history'].append(inference)
        pln_state['rules_applied'] += 1
        
        logger.info(f"Applied induction rule: {result.to_dict()}")
        
        return jsonify({
            'success': True,
            'inference': inference
        }), 200
        
    except Exception as e:
        logger.error(f"Error in induction inference: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/infer/abduction', methods=['POST'])
def infer_abduction():
    """Apply abduction inference rule"""
    try:
        data = request.get_json()
        
        if not data or 'premise1' not in data or 'premise2' not in data:
            return jsonify({
                'error': 'Missing required fields: premise1 and premise2'
            }), 400
        
        tv1 = TruthValue.from_dict(data['premise1'])
        tv2 = TruthValue.from_dict(data['premise2'])
        
        result = PLNInference.abduction(tv1, tv2)
        
        inference = {
            'rule': 'abduction',
            'premises': [tv1.to_dict(), tv2.to_dict()],
            'result': result.to_dict(),
            'timestamp': datetime.utcnow().isoformat()
        }
        
        pln_state['inference_history'].append(inference)
        pln_state['rules_applied'] += 1
        
        logger.info(f"Applied abduction rule: {result.to_dict()}")
        
        return jsonify({
            'success': True,
            'inference': inference
        }), 200
        
    except Exception as e:
        logger.error(f"Error in abduction inference: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/infer/conjunction', methods=['POST'])
def infer_conjunction():
    """Apply conjunction (AND) inference rule"""
    try:
        data = request.get_json()
        
        if not data or 'premise1' not in data or 'premise2' not in data:
            return jsonify({
                'error': 'Missing required fields: premise1 and premise2'
            }), 400
        
        tv1 = TruthValue.from_dict(data['premise1'])
        tv2 = TruthValue.from_dict(data['premise2'])
        
        result = PLNInference.conjunction(tv1, tv2)
        
        inference = {
            'rule': 'conjunction',
            'premises': [tv1.to_dict(), tv2.to_dict()],
            'result': result.to_dict(),
            'timestamp': datetime.utcnow().isoformat()
        }
        
        pln_state['inference_history'].append(inference)
        pln_state['rules_applied'] += 1
        
        logger.info(f"Applied conjunction rule: {result.to_dict()}")
        
        return jsonify({
            'success': True,
            'inference': inference
        }), 200
        
    except Exception as e:
        logger.error(f"Error in conjunction inference: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/infer/disjunction', methods=['POST'])
def infer_disjunction():
    """Apply disjunction (OR) inference rule"""
    try:
        data = request.get_json()
        
        if not data or 'premise1' not in data or 'premise2' not in data:
            return jsonify({
                'error': 'Missing required fields: premise1 and premise2'
            }), 400
        
        tv1 = TruthValue.from_dict(data['premise1'])
        tv2 = TruthValue.from_dict(data['premise2'])
        
        result = PLNInference.disjunction(tv1, tv2)
        
        inference = {
            'rule': 'disjunction',
            'premises': [tv1.to_dict(), tv2.to_dict()],
            'result': result.to_dict(),
            'timestamp': datetime.utcnow().isoformat()
        }
        
        pln_state['inference_history'].append(inference)
        pln_state['rules_applied'] += 1
        
        logger.info(f"Applied disjunction rule: {result.to_dict()}")
        
        return jsonify({
            'success': True,
            'inference': inference
        }), 200
        
    except Exception as e:
        logger.error(f"Error in disjunction inference: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/infer/negation', methods=['POST'])
def infer_negation():
    """Apply negation (NOT) inference rule"""
    try:
        data = request.get_json()
        
        if not data or 'premise' not in data:
            return jsonify({
                'error': 'Missing required field: premise'
            }), 400
        
        tv = TruthValue.from_dict(data['premise'])
        result = PLNInference.negation(tv)
        
        inference = {
            'rule': 'negation',
            'premises': [tv.to_dict()],
            'result': result.to_dict(),
            'timestamp': datetime.utcnow().isoformat()
        }
        
        pln_state['inference_history'].append(inference)
        pln_state['rules_applied'] += 1
        
        logger.info(f"Applied negation rule: {result.to_dict()}")
        
        return jsonify({
            'success': True,
            'inference': inference
        }), 200
        
    except Exception as e:
        logger.error(f"Error in negation inference: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/infer/revision', methods=['POST'])
def infer_revision():
    """Apply revision inference rule"""
    try:
        data = request.get_json()
        
        if not data or 'truth_value1' not in data or 'truth_value2' not in data:
            return jsonify({
                'error': 'Missing required fields: truth_value1 and truth_value2'
            }), 400
        
        tv1 = TruthValue.from_dict(data['truth_value1'])
        tv2 = TruthValue.from_dict(data['truth_value2'])
        
        result = PLNInference.revision(tv1, tv2)
        
        inference = {
            'rule': 'revision',
            'premises': [tv1.to_dict(), tv2.to_dict()],
            'result': result.to_dict(),
            'timestamp': datetime.utcnow().isoformat()
        }
        
        pln_state['inference_history'].append(inference)
        pln_state['rules_applied'] += 1
        
        logger.info(f"Applied revision rule: {result.to_dict()}")
        
        return jsonify({
            'success': True,
            'inference': inference
        }), 200
        
    except Exception as e:
        logger.error(f"Error in revision inference: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/history', methods=['GET'])
def get_history():
    """Get inference history"""
    try:
        limit = request.args.get('limit', default=100, type=int)
        
        history = pln_state['inference_history'][-limit:]
        
        return jsonify({
            'success': True,
            'count': len(history),
            'total_inferences': len(pln_state['inference_history']),
            'history': history
        }), 200
        
    except Exception as e:
        logger.error(f"Error getting history: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/history/clear', methods=['POST'])
def clear_history():
    """Clear inference history"""
    try:
        count = len(pln_state['inference_history'])
        pln_state['inference_history'].clear()
        pln_state['rules_applied'] = 0
        
        logger.info(f"Cleared inference history: {count} entries")
        
        return jsonify({
            'success': True,
            'message': f'Cleared {count} inference entries'
        }), 200
        
    except Exception as e:
        logger.error(f"Error clearing history: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/stats', methods=['GET'])
def get_stats():
    """Get PLN statistics"""
    try:
        rule_counts = {}
        for inference in pln_state['inference_history']:
            rule = inference['rule']
            rule_counts[rule] = rule_counts.get(rule, 0) + 1
        
        return jsonify({
            'success': True,
            'stats': {
                'total_inferences': len(pln_state['inference_history']),
                'rules_applied': pln_state['rules_applied'],
                'rule_distribution': rule_counts
            }
        }), 200
        
    except Exception as e:
        logger.error(f"Error getting stats: {str(e)}")
        return jsonify({'error': str(e)}), 500


if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8102))
    host = os.environ.get('HOST', '0.0.0.0')
    
    logger.info(f"Starting OpenCog PLN service on {host}:{port}")
    app.run(host=host, port=port, debug=False)
